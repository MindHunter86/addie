package balancer

import (
	"bytes"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/MindHunter86/anilibria-hlp-service/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog"
)

type ClusterBalancer struct {
	log *zerolog.Logger

	ulock    sync.RWMutex
	upstream *upstream

	sync.RWMutex
	size int
	ips  []*net.IP
}

func NewClusterBalancer(ctx context.Context) *ClusterBalancer {
	upstream := make(upstream)
	return &ClusterBalancer{
		log:      ctx.Value(utils.ContextKeyLogger).(*zerolog.Logger),
		upstream: &upstream,
	}
}

func (m *ClusterBalancer) BalanceByChunk(prefix, chunkname string) (_ string, server *BalancerServer, e error) {
	var key string
	if key, e = m.getKeyFromChunkName(&chunkname); e != nil {
		m.log.Debug().Err(e).Msgf("chunkname - '%s'; fallback to legacy balancing", chunkname)
		return
	}

	idx, e := strconv.Atoi(prefix + key)
	if e != nil {
		m.log.Debug().Err(e).Msgf("chunkname - '%s'; fallback to legacy balancing", chunkname)
		return
	}

	var ip *net.IP
	if ip = m.getServer(idx); ip == nil {
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

func (m *ClusterBalancer) getKeyFromChunkName(chunkname *string) (key string, e error) {
	if strings.Contains(*chunkname, "_") {
		key = strings.Split(*chunkname, "_")[1]
	} else if strings.Contains(*chunkname, "fff") {
		key = strings.ReplaceAll(*chunkname, "fff", "")
	} else {
		e = ErrUnparsableChunk
	}

	return
}

func (m *ClusterBalancer) getServer(idx int) (_ *net.IP) {
	if !m.TryRLock() {
		m.log.Debug().Msg("could not get lock for reading upstream; fallback to legacy balancing")
		return
	}
	defer m.RUnlock()

	if m.size == 0 {
		return
	}

	return m.ips[idx%int(m.size)]
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
		"Name", "Address", "Requests", "Last Request Time", "Is Down", "Status Time",
	})

	servers := m.upstream.getServers(&m.ulock)
	for _, server := range servers {
		tb.AppendRow([]interface{}{
			server.Name, server.Ip,
			server.handledRequests, server.lastRequestTime.String(),
			isDownHumanize(server.isDown), server.lastChanged.String(),
		})
	}

	tb.SortBy([]table.SortBy{
		{Number: 3, Mode: table.Dsc},
	})

	tb.Style().Options.SeparateRows = true

	return buf
}

func (m *ClusterBalancer) ResetStats() {
	m.upstream.resetServersStats(&m.ulock)
}

func (m *ClusterBalancer) ResetUpstream() {
	m.upstream.reset(&m.ulock)
}