package market

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jaysinco/Pgold/utils"
	"github.com/urfave/cli"
)

// ExportCmd run export subcommand
var ExportCmd = cli.Command{
	Name:  "export",
	Usage: "Export market data from database into file",
	Flags: []cli.Flag{
		utils.OutfileFlag,
		utils.StartDateFlag,
		utils.EndDateFlag,
		utils.OnlyTxOpenFlag,
	},
	Action: utils.InitWrapper(exportRun),
}

// ImportCmd run import subcommand
var ImportCmd = cli.Command{
	Name:  "import",
	Usage: "Import market data from file into database",
	Flags: []cli.Flag{
		utils.InfileFlag,
	},
	Action: utils.InitWrapper(importRun),
}

func exportRun(c *cli.Context) error {
	log.Println("run subcommand export")
	filename := c.String(utils.GetFlagName(utils.OutfileFlag))
	onlyTxOpen := c.Bool(utils.GetFlagName(utils.OnlyTxOpenFlag))
	start, err := utils.ParseDate(c.String(utils.GetFlagName(utils.StartDateFlag)))
	end, err := utils.ParseDate(c.String(utils.GetFlagName(utils.EndDateFlag)))
	if err != nil {
		return fmt.Errorf("export: wrong input date format: %v", err)
	}
	if err := exportMktData(filename, onlyTxOpen, start, end); err != nil {
		return fmt.Errorf("export: %v", err)
	}
	return nil
}

func exportMktData(filename string, onlyTxOpen bool, start, end time.Time) error {
	pgcs, err := utils.GetPriceFromDB(start, end, onlyTxOpen, true)
	if err != nil {
		return fmt.Errorf("query market data: %v", err)
	}
	if err := utils.WritePriceIntoBinFile(filename, pgcs); err != nil {
		return fmt.Errorf("write market data: %v", err)
	}
	log.Printf("%d records written\n", len(pgcs))
	return nil
}

func importRun(c *cli.Context) error {
	log.Println("run subcommand import")
	filename := c.String(utils.GetFlagName(utils.InfileFlag))
	if err := importMktData(filename, utils.DB); err != nil {
		return fmt.Errorf("import: %v", err)
	}
	return nil
}

func importMktData(filename string, db *sql.DB) error {
	pgcs, err := utils.GetPriceFromBinFile(filename)
	if err != nil {
		return fmt.Errorf("read market data: %v", err)
	}
	log.Printf("%d records readed", len(pgcs))
	success, err := utils.InsertPriceIntoDB(pgcs, true)
	if err != nil {
		return fmt.Errorf("insert market data: %v", err)
	}
	log.Printf("%d records inserted", success)
	return nil
}
