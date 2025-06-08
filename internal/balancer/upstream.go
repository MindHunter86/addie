package balancer

import (
	"net"
	"sort"
	"sync"
)

type upstream map[string]*BalancerServer

func (m upstream) getServer(l *sync.RWMutex, ip string) (server *BalancerServer, ok bool) {
	l.RLock()
	defer l.RUnlock()

	server, ok = m[ip]
	return
}

func (m upstream) putServer(l *sync.RWMutex, ip string, server *BalancerServer) {
	l.Lock()
	defer l.Unlock()

	m[ip] = server
}

func (m upstream) copy(l *sync.RWMutex) (_ map[string]*BalancerServer) {
	var buf = make(map[string]*BalancerServer)

	l.RLock()
	defer l.RUnlock()

	buf = m
	return buf
}

func (m upstream) resetServersStats(l *sync.RWMutex) {
	buf := m.copy(l)

	for _, server := range buf {
		server.resetStats()
	}
}

func (m upstream) getServers(l *sync.RWMutex) (servers []*BalancerServer) {
	buf := m.copy(l)

	for _, server := range buf {
		servers = append(servers, server)
	}

	return
}

func (m upstream) getIps(l *sync.RWMutex) (ips []*net.IP, _ int) {
	buf := m.copy(l)

	for _, server := range buf {
		ips = append(ips, &server.Ip)
	}

	sort.Slice(ips, func(i, j int) bool {
		return ips[i].String() < ips[j].String()
	})

	return ips, len(ips)
}
