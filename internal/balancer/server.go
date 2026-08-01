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

	sync.RWMutex
	isDown      bool
	lastChanged time.Time

	lastRequestTime time.Time
	handledRequests uint64

	log *zerolog.Logger
}

func newServer(l *zerolog.Logger, name string, ip *net.IP) *BalancerServer {
	if name[len(name)] == '.' {
		name = name[:len(name)-1]
	}

	return &BalancerServer{
		Name: name,
		Ip:   *ip,
		log:  l,
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

func (m *BalancerServer) monitor(interval, timeout time.Duration) {
	var e error
	var conn net.Conn

	for {
		if conn, e = net.DialTimeout("tcp", net.JoinHostPort(m.Ip.String(), "80"), timeout); e != nil {
			if !m.isDown {
				m.log.Info().Msgf("server %s was downed", m.Name)
				m.disable()
			}
		} else {
			if m.isDown {
				m.log.Info().Msgf("server %s was upped", m.Name)
				m.disable(false)
			}

			_ = conn.Close()
		}

		time.Sleep(interval)
	}
}
