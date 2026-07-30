package stats

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/util/conn"
	"github.com/rs/zerolog"
)

type graphite struct {
	prefix string

	conn *conn.Manager
	log  *zerolog.Logger

	sync.Mutex
}

func newGraphite(prefix, proto, server string, l *zerolog.Logger) (StatsWriter, error) {
	if prefix == "" || proto == "" || server == "" {
		return nil, errors.New("graphite server could not be empty")
	}

	g := &graphite{
		prefix: prefix + ".",
		conn:   conn.NewDefaultManager(proto, server, nil),

		log: l,
	}

	return g, nil
}

func (m *graphite) WriteOne(_ *time.Time, name string, val uint64) (int64, error) {
	m.Lock()
	defer m.Unlock()

	n, e := fmt.Fprintf(m.conn, "%s %d %d\n", m.prefix+name, val, time.Now().Unix())
	return int64(n), e
}

// func (m *graphite) WriteStringOne(name, val string) (int64, error) {
// 	m.Lock()
// 	defer m.Unlock()

// 	n, e := fmt.Fprintf(m.conn, "%s %s %d\n", m.prefix+name, val, time.Now().Unix())
// 	return int64(n), e
// }

// func (m *graphite) Write(values map[string]uint64) (cnt int64, e error) {
// 	m.Lock()
// 	defer m.Unlock()
// 	now := time.Now().Unix()

// 	for k, v := range values {
// 		n, err := fmt.Fprintf(m.conn, "%s %d %d\n", m.prefix+k, v, now)
// 		if err != nil {
// 			return
// 		}
// 		cnt += int64(n)
// 	}

// 	return
// }
