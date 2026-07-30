package stats

import "time"

type StatsWriter interface {
	// Write(map[string]uint64) (int64, error)
	WriteOne(*time.Time, string, uint64) (int64, error)
	// WriteStringOne(string, string)
}
