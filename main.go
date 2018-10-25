package main

import (
	"log"
	"os"

	"github.com/jaysinco/Pgold/utils"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Name = "pgold"
	app.Usage = "ICBC paper gold trader helping system"
	app.Flags = []cli.Flag{
		utils.ConfigFlag,
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:   "market",
			Usage:  "Fetch market data into database continuously",
			Action: utils.InitWrapper(marketRun),
		},
		cli.Command{
			Name:  "export",
			Usage: "Export market data from database into file",
			Flags: []cli.Flag{
				utils.OutfileFlag,
				utils.StartDateFlag,
				utils.EndDateFlag,
				utils.OnlyTxOpenFlag,
			},
			Action: utils.InitWrapper(exportRun),
		},
		cli.Command{
			Name:  "import",
			Usage: "Import market data from file into database",
			Flags: []cli.Flag{
				utils.InfileFlag,
			},
			Action: utils.InitWrapper(importRun),
		},
		cli.Command{
			Name:   "show",
			Usage:  "Show market history data through http server",
			Action: utils.InitWrapper(showRun),
		},
		cli.Command{
			Name:   "hint",
			Usage:  "Email trade tips continuously based on strategy",
			Action: utils.InitWrapper(hintRun),
		},
		cli.Command{
			Name:  "test",
			Usage: "Loopback test strategy using history data",
			Flags: []cli.Flag{
				utils.StartDateFlag,
				utils.EndDateFlag,
			},
			Action: utils.InitWrapper(testRun),
		},
		cli.Command{
			Name:  "batch",
			Usage: "Run serveral tasks simultaneously",
			Flags: []cli.Flag{
				utils.TaskListFlag,
			},
			Action: utils.InitWrapper(startRun),
		},
	}

	app.After = func(c *cli.Context) error {
		if utils.DB != nil {
			return utils.DB.Close()
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("pgold: %v", err)
	}
}
