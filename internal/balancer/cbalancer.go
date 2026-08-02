package balancer

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	braceexp "github.com/MindHunter86/addie/internal/utils/brace_exp"
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

func (m *ClusterBalancer) GetFQDNsByBrace(pattern string) ([]string, error) {
	return braceexp.Expand(pattern, 0)
}

func (m *ClusterBalancer) GetClusterName() string {
	switch m.cluster {
	case BalancerClusterNodes:
		return m.ccx.String("service-nodes")
	case BalancerClusterCloud:
		return m.ccx.String("service-cloud")
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

func (m *ClusterBalancer) BalanceByChunk(prefix, chunkname string, try int) (_ string, server *BalancerServer, e error) {
	var key string
	if key, e = m.getKeyFromChunkName(&chunkname); e != nil {
		m.log.Debug().Err(e).Msgf("chunkname - '%s'; fallback to legacy balancing", chunkname)
		return
	}

	var ip *net.IP
	idx0, idx1 := murmur3.Sum128([]byte(prefix + key))
	if ip = m.getServer(idx0, idx1, try); ip == nil {
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

// TODO String.Find - [X:]
func (*ClusterBalancer) getKeyFromChunkName(chunkname *string) (key string, e error) {
	if strings.Contains(*chunkname, "_") {
		key = strings.Split(*chunkname, "_")[1]
	} else if strings.Contains(*chunkname, "fff") {
		key = strings.ReplaceAll(*chunkname, "fff", "")
	} else {
		e = ErrUnparsableChunk
	}

	return
}

func (m *ClusterBalancer) getServer(idx1, idx2 uint64, try int) (ip *net.IP) {
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

	if idx0 = idx0 + uint64(try); idx0 > uint64(m.size) {
		idx0 = idx0 - uint64(m.size)
	}

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

func (m *ClusterBalancer) UpdateServersByFQDN(fqdns []string) {
	if len(fqdns) == 0 || fqdns[0] == "" {
		m.log.Error().Msgf("there no servers for updating %s cluster; check your args", m.GetClusterName())
		return
	}

	servers := make(map[string]net.IP)

	for _, f := range fqdns {
		if f == "" {
			m.log.Warn().Msgf("empty hostname detected in fqdns list, cluster %s", m.GetClusterName())
			continue
		}

		ips, _ := net.LookupIP(f)
		if len(ips) != 1 {
			m.log.Warn().Msg("abnormal IPs count in server by brace, check given hostnames; dropping server...")
			continue
		}

		m.log.Debug().Msgf("found server for %s : %s with %s", m.GetClusterName(), f, ips[0].String())
		servers[f] = ips[0]
	}

	m.log.Info().Msgf("updating cluster %s with %d servers", m.GetClusterName(), len(servers))
	m.UpdateServers(servers)
}

func (m *ClusterBalancer) UpdateServers(servers map[string]net.IP) {
	m.log.Trace().Msg("upstream servers debugging (I/II update iterations)")
	m.log.Info().Msg("[II] upstream update triggered")
	m.log.Trace().Interface("[II] servers", servers).Msg("")

	// find and append balancer's upstream
	for name, ip := range servers {
		if server, ok := m.upstream.getServer(&m.ulock, ip.String()); !ok {
			m.log.Trace().Msgf("[I] new server : %s", name)

			srv := newServer(m.log, name, &ip)
			m.upstream.putServer(&m.ulock, ip.String(), srv)

			// TODO - temporary, planned upgrade to BFD
			if ok = srv.healthcheck(400 * time.Millisecond); !ok {
				m.log.Warn().Msgf("updated serer %s in cluster %s is disabled due to tcp fails", srv.Name, m.GetClusterName())
				srv.disable()
			}

			m.log.Trace().Msgf("starting monitor for server %s", srv.Name)
			go srv.monitor(m.ccx.Duration("balancer-server-check-interval"), m.ccx.Duration("balancer-server-check-timeout"))
		} else {
			m.log.Trace().Msgf("[I] server found %s", name)
			server.disable(false)
		}
	}

	// find differs and disable dead servers
	// curr := m.upstream.copy(&m.ulock)
	// for _, server := range curr {
	// 	if _, ok := servers[server.Name]; !ok {
	// 		server.disable()
	// 		m.log.Trace().Msgf("[II] server - %s : disabled", server.Name)
	// 	} else {
	// 		m.log.Trace().Msgf("[II] server - %s : enabled", server.Name)
	// 	}
	// }

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
