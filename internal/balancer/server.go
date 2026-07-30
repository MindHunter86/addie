package balancer

import (
	"net"
	"sync"
	"time"
)

type BalancerServer struct {
	Ip   net.IP
	Name string

	sync.RWMutex
	isDown      bool
	lastChanged time.Time

	lastRequestTime time.Time
	handledRequests uint64
}

func newServer(name string, ip *net.IP) *BalancerServer {
	return &BalancerServer{
		Name: name,
		Ip:   *ip,
	}
}

func (m *BalancerServer) statRequest() {
	m.Lock()
	defer m.Unlock()

	m.lastRequestTime = time.Now()
	m.handledRequests++
}

func (m *BalancerServer) resetStats() {
	m.Lock()
	defer m.Unlock()

	m.lastRequestTime = time.Unix(0, 0)
	m.handledRequests = uint64(0)
}

func (m *BalancerServer) disable(disabled ...bool) {
	disabled = append(disabled, true)

	m.RLock()
	unchanged := m.isDown == disabled[0]
	m.RUnlock()

	if unchanged {
		return
	}

	m.Lock()
	defer m.Unlock()

	m.lastChanged = time.Now()
	m.isDown = disabled[0]
}
