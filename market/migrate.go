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
	filename := c.String(pg.FpComma(pg.OutfileFlag.Name))
	onlyTxOpen := c.Bool(pg.FpComma(pg.OnlyTxOpenFlag.Name))
	start, err := pg.ParseDate(c.String(pg.FpComma(pg.StartDateFlag.Name)))
	end, err := pg.ParseDate(c.String(pg.FpComma(pg.EndDateFlag.Name)))
	if err != nil {
		return fmt.Errorf("export: wrong input date format: %v", err)
	}
	log.Printf("[EXPORT] db(%s) -> file(%s)", pg.DBSTR, filename)
	if err := db2file(filename, start, end, onlyTxOpen); err != nil {
		return fmt.Errorf("export: %v", err)
	}
	return nil
}

// Import market price data
func Import(c *cli.Context) error {
	filename := c.String(pg.FpComma(pg.InfileFlag.Name))
	onlyTxOpen := c.Bool(pg.FpComma(pg.OnlyTxOpenFlag.Name))
	log.Printf("[IMPORT] db(%s) -> file(%s)", filename, pg.DBSTR)
	if err := file2db(filename, onlyTxOpen); err != nil {
		return fmt.Errorf("import: %v", err)
	}
	return nil
}

func db2file(filename string, start, end time.Time, onlyTxOpen bool) (err error) {
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
	defer func() {
		werr := fileWriter.Close()
		if werr != nil && err == nil {
			err = fmt.Errorf("close binary file writer: %v", werr)
		}
	}()
	return pipeTo(sqlReader, fileWriter, onlyTxOpen)
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
	sqlWriter, err := pg.NewSQLWriter()
	if err != nil {
		return fmt.Errorf("new sql writer: %v", err)
	}
	defer sqlWriter.Close()
	return pipeTo(fileReader, sqlWriter, onlyTxOpen)
}

func pipeTo(r pg.Reader, w pg.Writer, onlyTxOpen bool) error {
	length := r.Len()
	start, end := r.TimeRange()
	tmfmt := "06/01/02 15:04:05"
	log.Printf("[PIPETO] time range: %s -> %s", start.Format(tmfmt), end.Format(tmfmt))
	log.Printf("[PIPETO] %d records read", length)
	success := 0
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
	log.Printf("[PIPETO] %d records written", success)
	return nil
}
