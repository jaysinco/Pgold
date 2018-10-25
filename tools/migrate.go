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

// GetPriceFromDB collect price from database into list
func GetPriceFromDB(start, end time.Time, onlyTxOpen, print bool) (PriceList, error) {
	it, err := NewIterFromDatabase(start, end)
	if err != nil {
		return nil, fmt.Errorf("create new databse iterator: %v", err)
	}
	defer it.Close()
	total := it.Len()
	pgcs := make(PriceList, 0)
	count := 0
	for {
		data, dry := it.Next()
		if dry {
			break
		}
		if (!onlyTxOpen) || IsTxOpen(time.Unix(data.Timestamp, 0)) {
			pgcs = append(pgcs, data)
		}
		count++
		if print && count%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(count)/float32(total))
		}
	}
	if print {
		fmt.Print("\r")
	}
	return pgcs, nil
}

//InsertPriceIntoDB insert price into database
func InsertPriceIntoDB(pgcs PriceList, print bool) (success int, err error) {
	stmt, err := DB.Prepare("insert into pgmkt(txtime,bankbuy,banksell) values($1,$2,$3)")
	if err != nil {
		return 0, fmt.Errorf("prepare sql: %v", err)
	}
	for index, pgc := range pgcs {
		if print && index%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(index)/float32(len(pgcs))*100)
		}
		_, err = stmt.Exec(time.Unix(pgc.Timestamp, 0), pgc.Bankbuy, pgc.Banksell)
		if err != nil {
			continue
		} else {
			success++
		}
	}
	if print {
		fmt.Print("\r")
	}
	return success, nil
}

// GetPriceFromBinFile collect price from binary file into list
func GetPriceFromBinFile(filename string) (PriceList, error) {
	dfile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file '%s': %v", filename, err)
	}
	defer dfile.Close()
	var num int64
	err = binary.Read(dfile, binary.LittleEndian, &num)
	if err != nil {
		return nil, fmt.Errorf("read record num header: %v", err)
	}
	pgcs := make(PriceList, num)
	err = binary.Read(dfile, binary.LittleEndian, pgcs)
	if err != nil {
		return nil, fmt.Errorf("read records: %v", err)
	}
	return pgcs, nil
}

// WritePriceIntoBinFile write price from list into binary file
func WritePriceIntoBinFile(filename string, pgcs PriceList) error {
	dfile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file '%s': %v", filename, err)
	}
	defer dfile.Close()
	num := len(pgcs)
	if err := binary.Write(dfile, binary.LittleEndian, int64(num)); err != nil {
		return fmt.Errorf("write record num header: %v", err)
	}
	if err := binary.Write(dfile, binary.LittleEndian, pgcs); err != nil {
		return fmt.Errorf("write records: %v", err)
	}
	return nil
}
