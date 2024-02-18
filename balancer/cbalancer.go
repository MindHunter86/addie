package balancer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/MindHunter86/addie/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog"
	"github.com/spaolacci/murmur3"
	"github.com/urfave/cli/v2"
)

type ClusterBalancer struct {
	log     *zerolog.Logger
	ccx     *cli.Context
	runtime *runtime.Runtime

	cluster BalancerCluster

	ulock    sync.RWMutex
	upstream *upstream

	isDown bool

	sync.RWMutex
	size int
	ips  []*net.IP
}

func NewClusterBalancer(ctx context.Context, cluster BalancerCluster) *ClusterBalancer {
	upstream := make(upstream)

	return &ClusterBalancer{
		log:      ctx.Value(utils.ContextKeyLogger).(*zerolog.Logger),
		ccx:      ctx.Value(utils.ContextKeyCliContext).(*cli.Context),
		runtime:  ctx.Value(utils.ContextKeyRuntime).(*runtime.Runtime),
		cluster:  cluster,
		upstream: &upstream,
	}
}

func SetMaxTries(max uint) error {
	if max > 10 || max == 0 {
		return errors.New("balancer - max tries could not be more than 10 or == 0")
	}

	MaxTries = uint8(max)
	return nil
}

func (m *ClusterBalancer) GetClusterName() string {
	switch m.cluster {
	case BalancerClusterNodes:
		return m.ccx.String("consul-service-nodes")
	case BalancerClusterCloud:
		return m.ccx.String("consul-service-cloud")
	default:
		return ""
	}
}

func (m *ClusterBalancer) rLock(rel ...bool) (e error) {
	if rel = append(rel, false); rel[0] {
		m.RUnlock()
		return
	}

	if !m.TryRLock() {
		e = NewError(m, errLockMiss).SetFlag(IsRetriable)
	}
	return
}

func (m *ClusterBalancer) IsDown() (dwn bool, e error) {
	if e = m.rLock(); e != nil {
		return
	}
	defer m.rLock(true)

	dwn, e = m.isDown, NewError(m, errUpstreamIsDown).SetFlag(IsReroutable)
	return
}

func (m *ClusterBalancer) getChunkKey(chunkname *string) (key string, e error) {
	if strings.Contains(*chunkname, "_") {
		key = strings.Split(*chunkname, "_")[1]
	} else if strings.Contains(*chunkname, "fff") {
		key = strings.ReplaceAll(*chunkname, "fff", "")
	}

	if key = strings.TrimSpace(key); key == "" {
		e = NewErrorF(m, errUnparsableChunk, chunkname)
	}
	return
}

func (m *ClusterBalancer) getServer(idx1, idx2 uint64, coll uint8) (ip *net.IP, e error) {
	if e = m.rLock(); e != nil {
		return
	}
	defer m.rLock(true)

	if m.size == 0 {
		e = NewError(m, errUndefined+": m.size == 0").SetFlag(IsRetriable)
		return
	}

	// by default coll = 0, but if balancer receive errors: coll += 1 (limited by const MaxTries)
	// ? maybe `MaxTries^coll` is not needed; use `coll` only?
	idx0 := (idx1 % uint64(m.size)) + (idx2 % uint64(m.size)) + uint64(MaxTries^coll)
	ip = m.ips[idx0%uint64(m.size)]
	return
}

func (m *ClusterBalancer) BalanceByChunkname(prefix, chunkname string, try uint8) (_ string, server *BalancerServer, e error) {
	var dwn bool
	if dwn, e = m.IsDown(); dwn {
		return
	}

	var key string
	if key, e = m.getChunkKey(&chunkname); e != nil {
		return
	}

	var ip *net.IP
	idx1, idx2 := murmur3.Sum128([]byte(prefix + key))
	if ip, e = m.getServer(idx1, idx2, try); e != nil {
		return
	}

	var ok bool
	if server, ok = m.upstream.getServer(&m.ulock, ip.String()); !ok {
		e = NewError(m, errUndefined).SetFlag(IsReroutable)
	} else if server.isDown {
		e = NewErrorF(m, errServerIsDown, server.Name).SetFlag(IsBackupable)
	} else {
		server.statRequest()
	}

	return ip.String(), server, e
}

func (m *ClusterBalancer) UpdateServers(servers map[string]net.IP) {
	m.log.Trace().Msg("upstream servers debugging (I/II update iterations)")
	m.log.Info().Msg("[II] upstream update triggered")
	m.log.Trace().Interface("[II] servers", servers).Msg("")

	// find and append balancer's upstream
	for name, ip := range servers {
		if server, ok := m.upstream.getServer(&m.ulock, ip.String()); !ok {
			m.log.Trace().Msgf("[I] new server : %s", name)
			m.upstream.putServer(&m.ulock, ip.String(), newServer(name, &ip))
		} else {
			m.log.Trace().Msgf("[I] server found %s", name)
			server.disable(false)
		}
	}

	// find differs and disable dead servers
	curr, dwn := m.upstream.copy(&m.ulock), 0
	for _, server := range curr {
		if _, ok := servers[server.Name]; !ok {
			dwn++
			server.disable()
			m.log.Trace().Msgf("[II] server - %s : disabled", server.Name)
		} else {
			m.log.Trace().Msgf("[II] server - %s : enabled", server.Name)
		}
	}

	// update "balancer" (slice that used for getNextServer)
	m.Lock()
	defer m.Unlock()

	// !!!
	// !!!
	// !!!
	// !!!

	val, _, _ := m.runtime.Config.GetValue(runtime.ConfigParamA5bility)
	avail := val.(int)
	m.ips, m.size = m.upstream.getIps(&m.ulock)

	if (dwn != 0 && dwn*100/m.size > (100-avail)) || m.size == 0 {
		m.isDown = true
		m.log.Warn().Msgf("calc - (%d * 100 / %d), avail(calc) - %d, size - %d) cluster was marked as `offline`",
			dwn, m.size, 100-avail, m.size)
	} else if m.isDown {
		m.isDown = false
		m.log.Info().Msgf("calc - (%d * 100 / %d), avail(calc) - %d, size - %d) cluster was marked as `online`",
			dwn, m.size, 100-avail, m.size)
	}

	m.log.Trace().Interface("ips", m.ips).Msg("[II]")
	m.log.Trace().Interface("size", m.size).Msgf("[II]")
}

func (m *ClusterBalancer) GetStats() io.Reader {
	tb := table.NewWriter()
	defer tb.Render()

	isDownHumanize := func(i bool) string {
		switch i {
		case false:
			return "no"
		default:
			return "yes"
		}
	}

	buf := bytes.NewBuffer(nil)
	tb.SetOutputMirror(buf)
	tb.AppendHeader(table.Row{
		"Name", "Address", "Requests", "Last Diff", "First Diff", "Last Request Time", "Is Down", "Status Time",
	})

	servers := m.upstream.getServers(&m.ulock)
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].handledRequests > servers[j].handledRequests
	})

	round := func(val float64, precision uint) float64 {
		ratio := math.Pow(10, float64(precision))
		return math.Round(val*ratio) / ratio
	}

	for idx, server := range servers {
		var firstdiff, lastdiff float64

		if servers[0].handledRequests != 0 {
			firstdiff = (float64(server.handledRequests) * 100.00 / float64(servers[0].handledRequests)) - 100.00
		}

		if idx != 0 && servers[idx-1].handledRequests != 0 {
			lastdiff = (float64(server.handledRequests) * 100.00 / float64(servers[idx-1].handledRequests)) - 100.00
		}

		tb.AppendRow([]interface{}{
			server.Name, server.Ip,
			server.handledRequests, round(lastdiff, 2), round(firstdiff, 2), server.lastRequestTime.Format("2006-01-02T15:04:05.000"),
			isDownHumanize(server.isDown), server.lastChanged.Format("2006-01-02T15:04:05.000"),
		})
	}

	tb.AppendFooter(table.Row{"", "", "", "", "", "CLUSTER OFFLINE", isDownHumanize(m.isDown), ""})

	tb.Style().Options.SeparateRows = true

	return buf
}

func (m *ClusterBalancer) ResetStats() {
	m.upstream.resetServersStats(&m.ulock)
}

func (m *ClusterBalancer) ResetUpstream() {
	m.ulock.Lock()
	defer m.ulock.Unlock()

	upstream := make(upstream)
	m.upstream = &upstream
}
