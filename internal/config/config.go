package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/k0kubun/pp"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"github.com/valyala/bytebufferpool"
	"gopkg.in/yaml.v3"
)

type DynamicConfig struct {
	http *HttpClient
	wach *fsnotify.Watcher

	keys []string

	url  string
	path string
	temp string
	mxsz int64

	fint time.Duration
	fmtx sync.RWMutex
}

// func test() {
// 	cli := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)
// 	for _, fl := range cli.App.Flags {
// 		fl.Names()

// 	}

// 	for _, cat := range cli.App.VisibleFlagCategories() {
// 	}

// 	lala := dc.cli.Generic("1").(DynamicFlag[int])
// 	if abc := *lala.Load(); abc != 1 {
// 		panic("")
// 	}
// }

func NewDynamicConfig(c context.Context, catname string) (dc *DynamicConfig, e error) {
	dc = new(DynamicConfig)
	cli := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

	source := strings.TrimSpace(cli.String("dynamic-config-source"))

	// check if dynamic-config-source is url or file
	if isPath(source) {
		dc.path = source
	} else if isURL(source) {
		dc.url = source
	} else {
		return nil, errors.New("could not parse given dynamic-config-source; should be URL or path")
	}

	// check if dynamic-config-source-tmp is url or file
	tmpdir := cli.String("dynamic-config-source-tmp")
	if tmpdir != "" && !isPath(tmpdir) {
		return nil, errors.New("could not parse given dynamic-config-source-tmp; should be directory path")
	}

	// create temp file for url downloading
	if dc.url != "" {
		if dc.temp, e = getTmpFilePath(cli.App.Name, tmpdir); e != nil {
			return nil, utils.ExtraErrorWrapper(e, "could not create temp dir by dynamic-config-source-tmp;")
		}

		if dc.http, e = NewHttpClient(c, dc.url); e != nil {
			return
		}
	}

	if dc.keys = dc.lookupForConfigKeys(cli, catname); dc.keys == nil {
		return nil, errors.New("BUG: could not find dynamic config values in cli.Flags")
	}

	dc.fint = cli.Duration("dynamic-config-fetch-interval")
	dc.mxsz = cli.Int64("dynamic-config-max-size")

	return
}

func (m *DynamicConfig) onServiceBootstrap(c context.Context) (e error) {
	if m.wach, e = fsnotify.NewWatcher(); e != nil {
		return
	}

	log := utils.ContextValueExtract[*zerolog.Logger](c, utils.CtxZeroLogger)

	go func() {
	LOOP:
		for {
			select {
			case <-c.Done():
				break LOOP
			case e, ok := <-m.wach.Events:
				if !ok {
					log.Warn().Msg("could not get events from fsnotify Events goroutine, goroutine will be destroyed")
					return
				}

				log.Trace().Msg("caught fsnotify event")

				// !! TODO - 2DELETE
				pp.Print(e)

				if e.Has(fsnotify.Write) && e.Name == m.temp {
					log.Debug().Msg("temp file modification caught: updating dynamic config values...")
					// check md5 !
					// RELOAD
				}

			case err, ok := <-m.wach.Errors:
				if !ok {
					log.Warn().Msg("could not get events from fsnotify Errors goroutine, goroutine will be destroyed")
					return
				}
				log.Warn().Msg(utils.ExtraErrorWrapper(err, "caught fsnotify error").Error())
			}
		}
	}()

	return m.wach.Add(m.temp)
}

func (m *DynamicConfig) onServiceDestruct(_ context.Context) error {
	return m.wach.Close()
}

// Ticker function for donwloading Remote Source every
func (m *DynamicConfig) onServiceTicker1sec(c context.Context) (e error) {
	tick := utils.ContextValueExtract[uint64](c, utils.CtxTickerTick)
	if tick%uint64(m.fint.Seconds()) != 0 {
		return
	} else if !m.fmtx.TryLock() {
		return
	}

	defer m.fmtx.Unlock()
	return m.http.downloadSourceFromURL(m.url, m.temp)
}

func (*DynamicConfig) lookupForConfigKeys(c *cli.Context, catname string) (keys []string) {
	for _, cat := range c.App.VisibleFlagCategories() {
		if cat.Name() != catname {
			continue
		}

		if len(cat.Flags()) == 0 {
			continue
		}

		if keys == nil {
			keys = make([]string, 0, len(cat.Flags()))
		}

		for _, flag := range cat.Flags() {
			keys = append(keys, flag.Names()...)
		}
	}

	if len(keys) == 0 {
		return nil
	}

	return keys
}

// skipcq: SCC-U1000 temporary disabled
func (m *DynamicConfig) updateDynamicFlags() (e error) {
	// load new values
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	if buf.B, e = m.fetchContentFromFile(m.temp, buf.B); e != nil {
		return
	}

	var es *ExternalSource
	if es, e = m.unmarshalExternalSource(buf.B); e != nil {
		return
	}

	pp.Print(es)

	//

	// update config

	// update schema (regions + routes)
	return
}

// skipcq: SCC-U1000 temporary disabled
func (m *DynamicConfig) fetchContentFromFile(path string, buf []byte) (_ []byte, e error) {
	if path == "" {
		return nil, os.ErrNotExist
	}

	var fd *os.File
	if fd, e = os.Open(m.temp); e != nil {
		return
	}
	defer fd.Close()

	var fifo os.FileInfo
	if fifo, e = fd.Stat(); e != nil {
		return
	}

	size := fifo.Size()
	if size > m.mxsz {
		return nil, fmt.Errorf("rejecting file sized more than limit (size - %d; limit - %d)", size, m.mxsz)
	}

	if size < 0 || uint64(size)+1 > uint64(math.MaxInt) {
		return nil, fmt.Errorf("file is too large: %d bytes", size)
	}

	// increase capacity to minimize allocs

	if required := int(size) + 1; cap(buf) < required {
		buf = slices.Grow(buf, required-len(buf))
	}

	// based on os/file.go:791
	for {
		if len(buf) == cap(buf) {
			buf = slices.Grow(buf, 1)
		}

		n, err := fd.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]

		if err != nil {
			if err == io.EOF {
				return buf, nil
			}
			return buf, err
		}
	}
}

// skipcq: SCC-U1000 temporary disabled
func (m *DynamicConfig) unmarshalExternalSource(payload []byte) (_ *ExternalSource, _ error) {
	var es ExternalSource
	return &es, yaml.Unmarshal(payload, &es)
}
