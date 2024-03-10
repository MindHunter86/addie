package balancer

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand"
	"net"
	"sort"
	"sync"

	"github.com/MindHunter86/addie/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog"
	"github.com/spaolacci/murmur3"
	"github.com/urfave/cli/v2"
)

type ClusterBalancer struct {
	log *zerolog.Logger
	ccx *cli.Context

	cluster BalancerCluster

	ulock    sync.RWMutex
	upstream *upstream

	sync.RWMutex
	size int
	ips  []*net.IP
}

func NewClusterBalancer(ctx context.Context, cluster BalancerCluster) *ClusterBalancer {
	upstream := make(upstream)

	return &ClusterBalancer{
		log:      ctx.Value(utils.ContextKeyLogger).(*zerolog.Logger),
		ccx:      ctx.Value(utils.ContextKeyCliContext).(*cli.Context),
		cluster:  cluster,
		upstream: &upstream,
	}
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

func (m *ClusterBalancer) BalanceRandom() (_ string, server *BalancerServer, e error) {
	var ip *net.IP
	if ip = m.getRandomServer(); ip == nil {
		e = ErrUpstreamUnavailable
		return
	}

	server, ok := m.upstream.getServer(&m.ulock, ip.String())
	if !ok || server == nil {
		panic("balance result could not be find in balancer's upstream")
	} else if server.isDown {
		e = ErrServerUnavailable
	} else {
		server.statRequest()
	}

	return ip.String(), server, e
}

func (m *ClusterBalancer) BalanceByChunk(prefixbuf *bytes.Buffer, chunkname []byte) (_ string, server *BalancerServer, e error) {
	var key []byte
	if key, e = m.getKeyFromChunkName(chunkname); e != nil {
		m.log.Debug().Err(e).Msgf("chunkname - '%s'; fallback to legacy balancing", chunkname)
		return
	}

	prefixbuf.Write(key)

	var ip *net.IP
	if ip = m.getServer(murmur3.Sum128(prefixbuf.Bytes())); ip == nil {
		e = ErrUpstreamUnavailable
		return
	}

	server, ok := m.upstream.getServer(&m.ulock, ip.String())
	if !ok || server == nil {
		panic("balance result could not be find in balancer's upstream")
	} else if server.isDown {
		e = ErrServerUnavailable
	} else {
		server.statRequest()
	}

	return ip.String(), server, e
}

func (*ClusterBalancer) getKeyFromChunkName(chunkname []byte) (key []byte, _ error) {
	if bytes.Contains(chunkname, []byte("_")) {
		key = bytes.Split(chunkname, []byte("_"))[1]
	} else if bytes.Contains(chunkname, []byte("fff")) {
		key = bytes.ReplaceAll(chunkname, []byte("fff"), []byte(""))
	} else {
		return nil, ErrUnparsableChunk
	}

	return
}

func (m *ClusterBalancer) getServer(idx1, idx2 uint64) (ip *net.IP) {
	if !m.TryRLock() {
		m.log.Warn().Msg("could not get lock for reading upstream; fallback to legacy balancing")
		return
	}
	defer m.RUnlock()

	if m.size == 0 {
		return
	}

	idx3 := idx1 % uint64(m.size)
	idx4 := idx2 % uint64(m.size)
	idx0 := idx3 + idx4

	ip = m.ips[idx0%uint64(m.size)]
	return ip
}

func (m *ClusterBalancer) getRandomServer() (ip *net.IP) {
	if !m.TryRLock() {
		m.log.Error().Msg("could not get lock for reading upstream and force flag is false")
		return
	}
	defer m.RUnlock()

	if m.size == 0 {
		m.log.Error().Msg("could not get random server because of empty upstream")
		return
	}

	ip = m.ips[rand.Intn(m.size)] // skipcq: GSC-G404 math/rand is enough
	return
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
	curr := m.upstream.copy(&m.ulock)
	for _, server := range curr {
		if _, ok := servers[server.Name]; !ok {
			server.disable()
			m.log.Trace().Msgf("[II] server - %s : disabled", server.Name)
		} else {
			m.log.Trace().Msgf("[II] server - %s : enabled", server.Name)
		}
	}

	// update "balancer" (slice that used for getNextServer)
	m.Lock()
	defer m.Unlock()

	m.ips, m.size = m.upstream.getIps(&m.ulock)
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
