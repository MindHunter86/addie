package stats

import (
	"context"
	"errors"
	"expvar"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"go.uber.org/atomic"
)

type Stats struct { // skipcq: GO-W4013 new expose check skip
	incmetrics incMetrics
	avgmetrics avgMetrics
	permetrics persMetrics

	sync.Mutex
	rotateint  time.Duration
	rotatewarn time.Duration
	rotatelist []func()

	writers []StatsWriter

	queue chan *queueTask

	log *zerolog.Logger
}
type queueTask struct {
	mc  any
	val uint64
}

func NewStatsService(c context.Context) *Stats {
	cli := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

	gprefix := cli.String("stats-graphite-prefix")
	if utils.IsDevelVersion(cli.App.Version) && gprefix != "" {
		gprefix += "-dev"
	}

	st := &Stats{
		log: utils.ContextValueExtract[*zerolog.Logger](c, utils.CtxZeroLogger),

		incmetrics: make(incMetrics),
		avgmetrics: make(avgMetrics),
		permetrics: make(persMetrics),

		rotateint:  cli.Duration("stats-metrics-interval"),
		rotatewarn: cli.Duration("stats-metrics-loop-warn"),
		rotatelist: make([]func(), 0, _metricsEOL),

		writers: make([]StatsWriter, 0, 32),

		queue: make(chan *queueTask, cli.Int("stats-queue-buffer-size")),
	}

	if st.rotateint%(5*time.Second) != 0 {
		panic("application intervals must be multiples of 5 seconds (arg stats-metrics-interval invalid)")
	}

	if g, e := newGraphite(gprefix,
		cli.String("stats-graphite-proto"), cli.String("stats-graphite-server"), st.log); e != nil {
		st.log.Warn().Msg("graphite metric push was disable due to incorrect graphite params")
	} else {
		st.RegisterWriter(g)
	}

	for mn := range MetricNames {
		switch mnt := mn.(type) {
		case IncrementMetric:
			st.incmetrics[mnt] = atomic.NewUint64(0)
		case AverageMetric:

			st.avgmetrics[mnt] = &avgMetric{
				values: make([]*avgValue, 3+1),
				epoch:  atomic.NewUint32(1),
			}
			st.rotatelist = append(st.rotatelist, st.avgmetrics[mnt].rotate)
		case PersistentMetric:
			st.permetrics[mnt] = atomic.NewUint64(0)
		default:
			panic("BUG: undefined metric type detected on bootstrap process")
		}
	}

	return st
}

func (m *Stats) RegisterWriter(w StatsWriter) {
	m.writers = append(m.writers, w)
}

func (m *Stats) WriteIncMetric(mc IncrementMetric) {
	_ = m.incmetrics[mc].Inc()
}

func (m *Stats) WriteAvgMetric(mc AverageMetric, val uint64) {
	mcv := m.avgmetrics[mc]
	mcv.add(val, 0)
}

func (m *Stats) WriteAvgMetricFloat(mc AverageMetric, val float64, rounds uint) {
	if val <= 0 || rounds == 0 {
		return
	}

	mcv := m.avgmetrics[mc]
	mcv.add(uint64(val*float64(rounds)), rounds)
}

func (m *Stats) WritePersistentMetric(pm PersistentMetric, val uint64) {
	_ = m.permetrics[pm].Swap(val)
}

func (m *Stats) WritePersistentMetricInc(pm PersistentMetric) {
	_ = m.permetrics[pm].Inc()
}

func (m *Stats) WritePersistentMetricDec(pm PersistentMetric) {
	_ = m.permetrics[pm].Dec()
}

func (m *Stats) QueueWriteMetric(mc any, val ...uint64) {
	qt := acquireQueueTask()

	qt.mc, qt.val = mc, uint64(0)
	if len(val) != 0 {
		qt.val = val[0]
	}

	select {
	case m.queue <- qt:
	default:
		m.log.Warn().Msg("chan overflow detected in stats, metrics will be dropped")
		releaseQueueTask(qt)
		return
	}
}

func (m *Stats) onServiceTicker1sec(c context.Context) error {
	tick := utils.ContextValueExtract[uint64](c, utils.CtxTickerTick)
	if tick%uint64(m.rotateint.Seconds()) != 0 {
		return nil
	}

	if !m.TryLock() {
		m.WriteIncMetric(IMStatsLoopLockedError)
		return errors.New("could not start loop event because of mutex lock")
	}
	defer m.Unlock()

	lap := utils.ContextValueExtract[time.Time](c, utils.CtxTickerLap)

	m.WriteIncMetric(IMStatsLoopCount)

	for k, v := range m.incmetrics {
		// m.log.Trace().Msg("trying to push metric to graphite server INC metric " + MetricNames[k])
		m.forEachWriter(func(wr StatsWriter) {
			if _, e := wr.WriteOne(&lap, MetricNames[k], v.Load()); e != nil {
				m.log.Warn().Msg(utils.ExtraErrorWrapper(e, "writing incmetrics metrics...").Error())
			}
		})
	}

	for k, v := range m.avgmetrics {
		// m.log.Trace().Msg("trying to push metric to graphite server AVG metric " + MetricNames[k])
		m.forEachWriter(func(wr StatsWriter) {
			if _, e := wr.WriteOne(&lap, MetricNames[k], v.average()); e != nil {
				m.log.Warn().Msg(utils.ExtraErrorWrapper(e, "writing avgmetrics metrics...").Error())
			}
		})
	}

	for k, v := range m.permetrics {
		// m.log.Trace().Msg("trying to push metric to graphite server AVG metric " + MetricNames[k])
		m.forEachWriter(func(wr StatsWriter) {
			if _, e := wr.WriteOne(&lap, MetricNames[k], v.Load()); e != nil {
				m.log.Warn().Msg(utils.ExtraErrorWrapper(e, "writing permetrics metrics...").Error())
			}
		})
	}

	// add expvar export
	expvar.Do(func(kv expvar.KeyValue) {
		m.writeExpVars(&lap, kv.Value)
		// m.graphite.WriteStringOne("expvar."+kv.Key, kv.Value.String())
	})

	// rotate non-infinity metrics
	for _, fn := range m.rotatelist {
		fn()
	}

	if el := time.Since(lap); el >= m.rotatewarn {
		m.log.Warn().Msgf("stats loop took too much time %s", el.Round(time.Millisecond).String())
	}

	return nil
}

func (m *Stats) onServiceBootstrap(c context.Context) error {
	go func() {
	LOOP:
		for {
			select {
			case <-c.Done():
				break LOOP
			case qt, ok := <-m.queue:
				if ok {
					m.execQueueTask(qt)
				}
			}
		}
	}()
	return nil
}

func (m *Stats) forEachWriter(fn func(wr StatsWriter)) {
	if len(m.writers) == 0 {
		return
	}

	for _, wr := range m.writers {
		fn(wr)
	}
}

func (m *Stats) writeExpVars(t *time.Time, v any) {
	switch vt := v.(type) {
	case expvar.Func:
		m.writeExpVars(t, vt.Value())
	case runtime.MemStats:
		m.memStatsExporter(t, &vt)
	}
}

func (m *Stats) memStatsExporter(t *time.Time, ms *runtime.MemStats) {
	msv := reflect.ValueOf(*ms)
	mst := msv.Type()

	for i := 0; i < mst.NumField(); i++ {
		fl := msv.Field(i)
		if fl.Kind() != reflect.Uint64 && fl.Kind() != reflect.Uint32 {
			continue
		}

		m.forEachWriter(func(wr StatsWriter) {
			if _, e := wr.WriteOne(t, "expvars."+mst.Field(i).Name, fl.Uint()); e != nil {
				m.log.Warn().Msg(utils.ExtraErrorWrapper(e, "writing expvars metrics...").Error())
			}
		})
	}
}

func (m *Stats) execQueueTask(qt *queueTask) {
	defer releaseQueueTask(qt)

	mc, val := qt.mc, qt.val
	switch tmc := mc.(type) {
	case IncrementMetric:
		m.WriteIncMetric(tmc)
	case AverageMetric:
		m.WriteAvgMetric(tmc, val)
	default:
	}
}
