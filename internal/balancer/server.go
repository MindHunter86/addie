package balancer

import (
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type BalancerServer struct {
	Ip   net.IP
	Name string

	mu          sync.RWMutex
	isDown      bool
	lastChanged time.Time

	lastRequestTime time.Time
	handledRequests uint64

	log *zerolog.Logger
}

func newServer(l *zerolog.Logger, name string, ip *net.IP) *BalancerServer {
	return &BalancerServer{
		Name: name,
		Ip:   *ip,
		log:  l,
	}
}

func (m *BalancerServer) statRequest() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastRequestTime = time.Now()
	m.handledRequests++
}

func (m *BalancerServer) resetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastRequestTime = time.Unix(0, 0)
	m.handledRequests = uint64(0)
}

func (m *BalancerServer) disable(disabled ...bool) {
	disabled = append(disabled, true)

	m.mu.RLock()
	unchanged := m.isDown == disabled[0]
	m.mu.RUnlock()

	if unchanged {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastChanged = time.Now()
	m.isDown = disabled[0]
}

func (m *BalancerServer) monitor(interval, timeout time.Duration) {
	var ok bool

	for {
		ok = m.healthcheck(timeout)
		m.disable(!ok)

		time.Sleep(interval)
	}
}

func (m *BalancerServer) healthcheck(timeout time.Duration) (_ bool) {
	if conn, e := net.DialTimeout("tcp", net.JoinHostPort(m.Ip.String(), "80"), timeout); e == nil {
		return conn.Close() == nil
	}

	return
}
