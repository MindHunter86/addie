package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/urfave/cli/v2"

	"github.com/MindHunter86/addie/cmd/addie/flags"
	application "github.com/MindHunter86/addie/internal/app"
	"github.com/MindHunter86/addie/internal/utils"
)

func main() {
	// non-blocking writer
	dwr := diode.NewWriter(os.Stdout, 1000, 10*time.Millisecond, func(missed int) {
		fmt.Fprintf(os.Stderr, "diodes dropped %d messages; check your log-rate, please\n", missed)
	})

	// logger
	log := zerolog.New(zerolog.ConsoleWriter{
		Out: dwr,
	}).With().Timestamp().Caller().Logger()

	zerolog.CallerMarshalFunc = callerMarshalFunc
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	// application
	app := cli.NewApp()
	cli.VersionFlag = &cli.BoolFlag{
		Name:               "version",
		Usage:              "show version",
		Aliases:            []string{"V"},
		DisableDefaultText: true,
	}
	cli.VersionPrinter = func(_ *cli.Context) {
		fmt.Printf("%s\t%s\n", version, buildtime)
	}

	app.Version, app.Name, app.Usage, app.Copyright = version, name, usage, copyright
	app.Authors = []*cli.Author{{
		Name:  "MindHunter86",
		Email: "mindhunter86@vkom.cc",
	}}

	app.HideHelpCommand = true
	app.Flags = flags.FlagsInitialization(
		!strings.Contains(strings.Join(os.Args, " "), "--expert-mode"), app.Name)

	app.Action = func(c *cli.Context) (e error) {
		var lvl zerolog.Level
		if lvl, e = zerolog.ParseLevel(c.String("log-level")); e != nil {
			return utils.ExtraErrorWrapper(e, "parsing logger level")
		}
		zerolog.SetGlobalLevel(lvl)

		var syslogWriter = io.Discard
		if len(c.String("syslog-server")) != 0 {
			if runtime.GOOS == "windows" {
				log.Error().Msg("sorry, but syslog is not worked for windows; golang does not support syslog for win systems")
				return utils.ExtraErrorWrapper(os.ErrProcessDone, "detect windows OS and syslog flag together")
			}
			log.Debug().Msg("connecting to syslog server ...")

			if syslogWriter, e = utils.SetUpSyslogWriter(c); e != nil {
				return utils.ExtraErrorWrapper(e, "setup syslog writer")
			}
			log.Debug().Msg("syslog connection established; reset zerolog for MultiLevelWriter set ...")

			log = zerolog.New(zerolog.MultiLevelWriter(
				zerolog.ConsoleWriter{Out: dwr},
				syslogWriter,
			)).With().Timestamp().Caller().Logger()

			log.Info().Msg("zerolog reinitialized; starting app...")
		}

		// localbuilded versions tests
		defer func() {
			if version != "localbuilded" {
				return
			}

			if r := recover(); r != nil {
				fmt.Println("Recovered. Error:\n", r)
				fmt.Println(string(debug.Stack()))
				os.Exit(1)
			}
		}()

		log.Debug().Msgf("%s (%s) builded %s now is ready, starting main service...", app.Name, version, buildtime)
		return application.NewApp(c, &log, syslogWriter).Bootstrap()
	}

	// * sort.Sort of Flags uses too much allocs; temporary disabled
	//
	// sort.Sort(cli.FlagsByName(app.Flags))
	// sort.Sort(cli.CommandsByName(app.Commands))

	// run configured application
	var retcode int
	if e := app.Run(os.Args); e != nil {
		log.WithLevel(zerolog.FatalLevel).Msg(e.Error())
		retcode = 1
	}

	// TODO avoid this
	// diode hasn't Wait() method, so we need to use this `250` shit
	// fmt.Println("waiting for diode buf")
	// time.Sleep(250 * time.Millisecond)
	if e := dwr.Close(); e != nil {
		fmt.Fprintf(os.Stderr, utils.ExtraErrorWrapper(e, "closing diode buffer").Error())
	}

	cli.OsExiter(retcode)
}

func callerMarshalFunc(_ uintptr, file string, line int) string {
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short
	return file + ":" + strconv.Itoa(line)
}
