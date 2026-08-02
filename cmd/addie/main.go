package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MindHunter86/addie/cmd/addie/flags"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/urfave/cli/v2"

	application "github.com/MindHunter86/addie/internal/app"
)

func main() {
	// application
	app := cli.NewApp()
	cli.VersionFlag = &cli.BoolFlag{
		Name:               "version",
		Usage:              "show version",
		Aliases:            []string{"V"},
		DisableDefaultText: true,
	}
	cli.VersionPrinter = func(*cli.Context) {
		fmt.Printf("%s\t%s\n", version, buildtime)
	}

	app.Version, app.Name, app.Usage, app.Copyright = version, name, usage, copyright
	app.Authors = []*cli.Author{{
		Name:  "MindHunter86",
		Email: "mindhunter86@vkom.cc",
	}}

	app.HideHelpCommand = true
	app.Flags = flags.FlagsInitialization(
		!strings.Contains(strings.Join(os.Args, " "), "--expert-mode"))

	app.Action = func(c *cli.Context) (e error) {
		// logger v2
		var ulog *utils.Logger
		if ulog, e = utils.NewLogger(c); e != nil {
			return
		}
		defer func() {
			// * diode hasn't Wait() method, so we need to use this `150` shit
			time.Sleep(150 * time.Millisecond)

			if err := ulog.Destroy(); err != nil {
				fmt.Fprint(os.Stderr, utils.ExtraErrorWrapper(err, "logger destroy").Error())
			}
		}()

		// TODO - add access logger from saeko
		// log, alog := ulog.SystemLogger(), ulog.AccessLogger()
		log, _ := ulog.SystemLogger(), ulog.AccessLogger()

		// localbuilded versions tests
		defer func() {
			if r := recover(); r != nil {
				stck := string(debug.Stack())
				log.Error().Msg("Main Panic recovered: " + stck)

				fmt.Println("Recovered. Error:\n", r)
				fmt.Println(stck)

				os.Exit(1)
			}
		}()

		log.Debug().Msgf("%s (%s) builded %s now is ready, starting main service...",
			app.Name, version, buildtime)

		// return service.NewService(c, log, alog).Bootstrap()
		return application.NewApp(c, log, nil).Bootstrap()
	}

	// run configured application
	var exitcode int
	if e := app.Run(os.Args); e != nil {
		fmt.Fprintln(os.Stderr, utils.ExtraErrorWrapper(e, "running application").Error())
		exitcode = 1
	}

	cli.OsExiter(exitcode)
}
