package stats

import (
	"math"

	"go.uber.org/atomic"
)

type avgValue struct {
	sum, cnt *atomic.Uint64
	scale    uint
}
type avgMetric struct {
	// values snapshots
	values []*avgValue

	// value in range of 1..3
	epoch *atomic.Uint32
}
type incMetrics map[IncrementMetric]*atomic.Uint64
type avgMetrics map[AverageMetric]*avgMetric
type persMetrics map[PersistentMetric]*atomic.Uint64

func (m *avgMetric) average() uint64 {
	idx := m.epoch.Load()

	// get last epoch
	idx--
	if idx == 0 {
		idx = 3
	}

	val := m.getValue(idx, 0)
	if val.sum.Load() == 0 || val.cnt.Load() == 0 {
		return 0
	}

	if val.scale != 0 {
		return uint64(math.Floor((float64(val.sum.Load()) / float64(val.scale)) / float64(val.cnt.Load())))
	}

	return uint64(math.Floor(float64(val.sum.Load()) / float64(val.cnt.Load())))
}

func (m *avgMetric) rotate() {
	idx := m.epoch.Load()
	defer func() {
		_ = m.epoch.Swap(idx)
	}()

	// prepare next epoch
	if idx == 3 {
		idx = 0
	}
	idx++

	// reset values for next value store
	val := m.getValue(idx, 0)
	val.cnt.Store(0)
	val.sum.Store(0)
}

func (m *avgMetric) add(v uint64, scl uint) {
	idx := m.epoch.Load()
	val := m.getValue(idx, scl)

	_ = val.cnt.Inc()
	_ = val.sum.Add(v)
}

func (m *avgMetric) getValue(idx uint32, scl uint) *avgValue {
	if m.values[idx] == nil {
		m.values[idx] = &avgValue{
			cnt:   atomic.NewUint64(0),
			sum:   atomic.NewUint64(0),
			scale: scl,
		}
	}
	return m.values[idx]
}
