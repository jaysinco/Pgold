package market

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// Export market price data
func Export(c *cli.Context) error {
	log.Println("[EXPORT] run")
	filename := c.String(pg.FpComma(pg.OutfileFlag.Name))
	onlyTxOpen := c.Bool(pg.FpComma(pg.OnlyTxOpenFlag.Name))
	start, err := pg.ParseDate(c.String(pg.FpComma(pg.StartDateFlag.Name)))
	end, err := pg.ParseDate(c.String(pg.FpComma(pg.EndDateFlag.Name)))
	if err != nil {
		return fmt.Errorf("export: wrong input date format: %v", err)
	}
	if err := db2file(filename, start, end, onlyTxOpen); err != nil {
		return fmt.Errorf("export: %v", err)
	}
	return nil
}

// Import market price data
func Import(c *cli.Context) error {
	log.Println("[IMPORT] run")
	filename := c.String(pg.FpComma(pg.InfileFlag.Name))
	onlyTxOpen := c.Bool(pg.FpComma(pg.OnlyTxOpenFlag.Name))
	if err := file2db(filename, onlyTxOpen); err != nil {
		return fmt.Errorf("import: %v", err)
	}
	return nil
}

func db2file(filename string, start, end time.Time, onlyTxOpen bool) error {
	sqlReader, err := pg.NewSQLReader(start, end)
	if err != nil {
		return fmt.Errorf("new sql reader: %v", err)
	}
	defer sqlReader.Close()
	fh, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file '%s': %v", filename, err)
	}
	defer fh.Close()
	fileWriter, err := pg.NewBinaryWriter(fh)
	if err != nil {
		return fmt.Errorf("new binary file writer: %v", err)
	}
	defer fileWriter.Close()
	return transfer(sqlReader, fileWriter, onlyTxOpen)
}

func file2db(filename string, onlyTxOpen bool) error {
	fh, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file '%s': %v", filename, err)
	}
	defer fh.Close()
	fileReader, err := pg.NewBinaryReader(fh)
	if err != nil {
		return fmt.Errorf("new binary file reader: %v", err)
	}
	defer fileReader.Close()
	sqlWriter := pg.NewSQLWriter()
	defer sqlWriter.Close()
	return transfer(fileReader, sqlWriter, onlyTxOpen)
}

func transfer(r pg.Reader, w pg.Writer, onlyTxOpen bool) error {
	length := r.Len()
	success := 0
	log.Printf("%d records read\n", length)
	for i := 0; i < length; i++ {
		p := new(pg.Price)
		if _, err := r.Read(p); err != nil {
			return fmt.Errorf("read price: %v", err)
		}
		if (!onlyTxOpen) || pg.IsTradeOpen(time.Unix(p.Timestamp, 0)) {
			if err := w.Write(p); err == nil {
				success++
			}
		}
		if i%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(i+1)/float32(length)*100)
		}
	}
	fmt.Print("\r")
	log.Printf("%d records written\n", success)
	return nil
}
