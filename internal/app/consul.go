package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/MindHunter86/addie/internal/balancer"
	"github.com/MindHunter86/addie/internal/runtime"
	"github.com/MindHunter86/addie/utils"
	capi "github.com/hashicorp/consul/api"
	"github.com/rs/zerolog"
)

var defaultOpts = &capi.QueryOptions{
	AllowStale:        true,
	UseCache:          false,
	RequireConsistent: false,
}

type consulClient struct {
	*capi.Client
	ctx context.Context

	balancers []balancer.Balancer
}

var (
	errConsulInvalidCluster = errors.New("clustername cound not be empty")
)

func newConsulClient(balancers ...balancer.Balancer) (client *consulClient, e error) {
	cfg := capi.DefaultConfig()

	cfg.Address = gCli.String("consul-address")
	if cfg.Address == "" {
		gLog.Warn().Msg("given consul address could not be empty")
		return nil, errors.New("given consul address could not be empty")
	}

	if gCli.String("consul-service-nodes") == "" || gCli.String("consul-service-cloud") == "" {
		gLog.Warn().Msg("given consul services could not be empty")
		return nil, errors.New("given consul services name could not be empty")
	}

	for _, blcnr := range balancers {
		if blcnr.GetClusterName() == "" {
			e = errConsulInvalidCluster
			return
		}
	}

	client = new(consulClient)
	client.Client, e = capi.NewClient(cfg)
	client.balancers = balancers
	return
}

func (m *consulClient) bootstrap() {
	gLog.Debug().Msg("consul bootrap started")

	var eventDone context.CancelFunc
	m.ctx, eventDone = context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var errs = make(chan error, 16)

	// goroutine helper - consul health
	listenClusterEvents := func(wait *sync.WaitGroup, ec chan error, payload func() error) {
		wait.Add(1)

		go func(done func(), gopayload func() error, errors chan error) {
			errors <- gopayload()
			done()
		}(wait.Done, payload, ec)
	}

	// goroutine helper - consul KV
	listenClusterKVs := func(wait *sync.WaitGroup, payload func()) {
		wait.Add(1)
		gLog.Debug().Msg("starting consul config watchdog")

		go func(done, gopayload func()) {
			gopayload()
			done()
		}(wait.Done, payload)
	}

	// consul health service watchdog
	for _, clusterBalancer := range m.balancers {
		cbalancer := clusterBalancer
		listenClusterEvents(&wg, errs, func() error {
			return m.listenClusterEvents(cbalancer)
		})
	}

	// consul KV watchdog
	runpatch := gCtx.Value(utils.ContextKeyRPatcher).(chan *runtime.RuntimePatch)
	listenClusterKVs(&wg, func() {
		m.configKeyWatchdog(runpatch)
	})

loop:
	for {
		select {
		case err := <-errs:
			if err == nil {
				continue
			}

			gLog.Error().Err(err).Msg("")
			break loop
		case <-gCtx.Done():
			gLog.Debug().Msg("internal abort() has been caught")
			break loop
		}
	}

	eventDone()
	gLog.Debug().Msg("waiting for goroutines")
	wg.Wait()
}

func (m *consulClient) listenClusterEvents(cluster balancer.Balancer) (e error) {
	gLog.Debug().Msgf("consul event listener started for cluster %s", cluster.GetClusterName())
	defer gLog.Debug().Msgf("consul event listener stopped for cluster %s", cluster.GetClusterName())

	var idx uint64
	var servers map[string]net.IP
	var fails uint8

	for {
		if gCtx.Err() != nil {
			return
		}

		if fails > uint8(3) && !gCli.Bool("consul-ignore-errors") {
			gLog.Error().Msg("too many unsuccessfully tempts of serverlist receiving")
			return
		}

		if servers, idx, e = m.getHealthServers(idx, cluster.GetClusterName()); errors.Is(e, context.Canceled) {
			return
		} else if e != nil {
			gLog.Warn().Uint8("fails", fails).Err(e).
				Msg("there some problems with serverlist receiving from consul")
			fails = fails + 1

			time.Sleep(1 * time.Second)
			continue
		}

		gLog.Debug().Msg("consul listenEvents iteration triggered")
		fails = 0

		if len(servers) == 0 {
			gLog.Warn().Msg("received empty serverlist from consul")
		}

		if gLog.GetLevel() == zerolog.TraceLevel {
			gLog.Trace().Msg("received serverlist debug")

			for _, ip := range servers {
				gLog.Trace().Msgf("received serverlist entry - %s", ip.String())
			}
		}

		cluster.UpdateServers(servers)
	}
}

// func (m *consulClient)

func (m *consulClient) getHealthServers(idx uint64, service string) (_ map[string]net.IP, _ uint64, e error) {
	opts := *defaultOpts
	opts.WaitIndex = idx

	entries, meta, e := m.Health().Service(service, "", true, opts.WithContext(m.ctx))
	if e != nil {
		return nil, idx, e
	}

	var ip net.IP
	var servers = make(map[string]net.IP)

	for _, entry := range entries {
		gLog.Debug().Msgf("new health service entry %s:%d", entry.Node.Address, entry.Service.Port)

		ip = net.ParseIP(entry.Node.Address)
		if ip == nil {
			gLog.Warn().Msgf("there is invalid server address from consul - %s", entry.Node.Address)
			continue
		}

		servers[entry.Node.Node] = ip
	}

	return servers, meta.LastIndex, e
}

func (m *consulClient) updateLimiterSwitcher(enabled string) (e error) {
	kv := &capi.KVPair{}
	kv.Key, kv.Value = m.getPrefixedSettingsKey(utils.CfgLimiterSwitcher), []byte(enabled)

	_, e = m.KV().Put(kv, nil)
	return e
}

func (m *consulClient) updateQualityRewrite(q utils.TitleQuality) (e error) {
	kv, buf := &capi.KVPair{}, bytes.NewBufferString(q.String())
	kv.Key, kv.Value = m.getPrefixedSettingsKey(utils.CfgQualityLevel), buf.Bytes()

	_, e = m.KV().Put(kv, nil)
	return
}

func (*consulClient) getPrefixedSettingsKey(key string) string {
	return fmt.Sprintf("%s/settings/%s", gCli.String("consul-kv-prefix"), key)
}

func (m *consulClient) configKeyWatchdog(runpatch chan *runtime.RuntimePatch) {
	var idx uint64
	opts, prefix := *defaultOpts, m.getPrefixedSettingsKey("")

	timeCooler := func() { time.Sleep(5 * time.Second) }

loop:
	for {
		select {
		case <-m.ctx.Done():
			break loop
		default:
			opts.WaitIndex = idx
			pairs, meta, e := m.KV().List(prefix, opts.WithContext(m.ctx))

			if errors.Is(e, context.Canceled) {
				break loop
			} else if e != nil {
				gLog.Error().Err(e).Msgf("could not get consul values for %s prefix", prefix)
				timeCooler()
				continue
			} else if len(pairs) == 0 {
				gLog.Warn().Msg("consul sent empty values")
				timeCooler()
				continue
			}

			for _, kvpair := range pairs {
				if kvpair == nil {
					gLog.Warn().Msg("empty value detected in kvpairs from consul response")
					continue
				}

				pathkey := strings.Split(kvpair.Key, "/")
				patchkey := pathkey[len(pathkey)-1]

				ptype, ok := runtime.RuntimeUtilsBindings[patchkey]
				if !ok {
					gLog.Warn().Msgf("consul key %s not found in runtime bindings", patchkey)
					continue
				}

				gLog.Debug().
					Msgf("found key %s in runtime bindings, applying runtime patch", patchkey)

				// default patch
				patch := &runtime.RuntimePatch{
					Type:  ptype,
					Patch: kvpair.Value,
				}

				runpatch <- patch
			}

			idx = meta.LastIndex
		}
	}
}
