package market

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"os"
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
		utils.OnlyTxOpenFlag,
	},
	Action: utils.InitWrapper(exportRun),
}

func exportRun(c *cli.Context) error {
	filename := c.String(utils.OutfileFlag.Name)
	if filename == "" {
		return fmt.Errorf("output file name must be given")
	}
	log.Printf("start exporting market data into '%s'", filename)

	if err := exportMktData(filename, c.Bool(utils.OnlyTxOpenFlag.Name), utils.DB); err != nil {
		return fmt.Errorf("export market data: %v", err)
	}
	return nil
}

func exportMktData(filename string, onlyTxOpen bool, db *sql.DB) error {
	dfile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file '%s': %v", filename, err)
	}
	defer dfile.Close()
	pgcs, err := queryMktData(db, onlyTxOpen)
	if err != nil {
		return fmt.Errorf("query market data: %v", err)
	}
	num := len(pgcs)
	if err := binary.Write(dfile, binary.LittleEndian, int64(num)); err != nil {
		return fmt.Errorf("write record num header: %v", err)
	}
	if err := binary.Write(dfile, binary.LittleEndian, pgcs); err != nil {
		return fmt.Errorf("write records: %v", err)
	}
	log.Printf("%d records written\n", num)
	return nil
}

func queryMktData(db *sql.DB, onlyTxOpen bool) ([]pgprice, error) {
	rows, err := db.Query(`SELECT cast(row_number() OVER (ORDER BY txtime) as float)/(SELECT count(*) FROM pgmkt) * 100 percent, 
	                              txtime, bankbuy, banksell FROM pgmkt ORDER BY txtime`)
	if err != nil {
		return nil, fmt.Errorf("query data from table 'pgmkt': %v", err)
	}
	defer rows.Close()
	txtime := new(time.Time)
	pgcs := make([]pgprice, 0)
	count := 0
	percent := 0.0
	for rows.Next() {
		var pgc pgprice
		if err := rows.Scan(&percent, txtime, &pgc.Bankbuy, &pgc.Banksell); err != nil {
			return nil, fmt.Errorf("scan rows: %v", err)
		}
		if (!onlyTxOpen) || utils.IsTxOpen(txtime) {
			pgc.Timestamp = txtime.Unix()
			pgcs = append(pgcs, pgc)
		}
		count++
		if count%100 == 0 {
			fmt.Printf("\r >> %.1f%%", percent)
		}
	}
	fmt.Print("\r")
	return pgcs, nil
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

func importRun(c *cli.Context) error {
	filename := c.String(utils.InfileFlag.Name)
	if filename == "" {
		return fmt.Errorf("input file name must be given")
	}
	log.Printf("start importing market data from '%s'", filename)

	if err := importMktData(filename, utils.DB); err != nil {
		return fmt.Errorf("import market data: %v", err)
	}
	return nil
}

func importMktData(filename string, db *sql.DB) error {
	pgcs, err := readMktData(filename)
	if err != nil {
		return fmt.Errorf("read market data from '%s': %v", filename, err)
	}
	log.Printf("%d records readed", len(pgcs))
	stmt, err := db.Prepare("insert into pgmkt(txtime,bankbuy,banksell) values($1,$2,$3)")
	if err != nil {
		return fmt.Errorf("prepare sql: %v", err)
	}
	count := 0
	for index, pgc := range pgcs {
		if index%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(index)/float32(len(pgcs))*100)
		}
		_, err = stmt.Exec(time.Unix(pgc.Timestamp, 0), pgc.Bankbuy, pgc.Banksell)
		if err != nil {
			continue
		} else {
			count++
		}
	}
	fmt.Print("\r")
	log.Printf("%d records inserted", count)
	return nil
}

func readMktData(filename string) ([]pgprice, error) {
	dfile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file '%s': %v", filename, err)
	}
	var num int64
	err = binary.Read(dfile, binary.LittleEndian, &num)
	if err != nil {
		return nil, fmt.Errorf("read record num header: %v", err)
	}
	pgcs := make([]pgprice, num)
	err = binary.Read(dfile, binary.LittleEndian, pgcs)
	if err != nil {
		return nil, fmt.Errorf("read records: %v", err)
	}
	return pgcs, nil
}

type pgprice struct {
	Timestamp int64
	Bankbuy   float32
	Banksell  float32
}

func (p pgprice) String() string {
	tm := time.Unix(p.Timestamp, 0)
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d | %.2f | %.2f",
		tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), p.Bankbuy, p.Banksell)
}
