package market

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
	if err := db2file(filename, start, end, onlyTxOpen, true); err != nil {
		return fmt.Errorf("export: %v", err)
	}
	return nil
}

// Import market price data
func Import(c *cli.Context) error {
	filename := c.String(pg.FpComma(pg.InfileFlag.Name))
	onlyTxOpen := c.Bool(pg.FpComma(pg.OnlyTxOpenFlag.Name))
	start, err := pg.ParseDate(c.String(pg.FpComma(pg.StartDateFlag.Name)))
	end, err := pg.ParseDate(c.String(pg.FpComma(pg.EndDateFlag.Name)))
	if err != nil {
		return fmt.Errorf("import: wrong input date format: %v", err)
	}
	log.Printf("[IMPORT] db(%s) -> file(%s)", filename, pg.DBSTR)
	if err := file2db(filename, start, end, onlyTxOpen); err != nil {
		return fmt.Errorf("import: %v", err)
	}
	return nil
}

// Autosave market price data
func Autosave(c *cli.Context) error {
	log.Println("[AUTOSAVE] run")

	savedir, _ := filepath.Abs(pg.Config.Autosave.Savedir)
	savedir = filepath.ToSlash(savedir)
	if fi, err := os.Stat(savedir); err != nil || !fi.IsDir() {
		return fmt.Errorf("autosave: '%s' not exist/directory", savedir)
	}
	log.Printf("[AUTOSAVE] save directory is '%s'", savedir)

	autoMatcher := regexp.MustCompile(`pg~\d\d\d\d\d\d.dat`)
	for {
		savename := "pg~" + time.Now().Add(-24*time.Hour).Format("060102") + ".dat"
		filename := savedir + "/" + savename
		if _, err := os.Stat(filename); err != nil && os.IsNotExist(err) {
			start, _ := pg.ParseDate(pg.StartDateFlag.Value)
			end, _ := pg.ParseDate(time.Now().Format("060102"))
			if err := db2file(filename, start, end, false, false); err != nil {
				return fmt.Errorf("autosave: db->file: %v", err)
			}
		} else {
			// to-write file already exist!
		}
		files, _ := ioutil.ReadDir(savedir)
		for _, f := range files {
			nm := f.Name()
			if autoMatcher.MatchString(nm) && nm != savename {
				del := savedir + "/" + f.Name()
				if err := os.Remove(del); err != nil {
					return fmt.Errorf("autosave: remove '%s': %v", del, err)
				}
			}
		}

		tomorrow, _ := pg.ParseDate(time.Now().Add(24 * time.Hour).Format("060102"))
		awake := tomorrow.Add(time.Duration(pg.Config.Autosave.Hour * float32(time.Hour)))
		time.Sleep(time.Until(awake))
	}
}

func db2file(filename string, start, end time.Time, onlyTxOpen bool, printLog bool) (err error) {
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
	return pipeTo(sqlReader, fileWriter, func(p *pg.Price) bool {
		return !onlyTxOpen || pg.IsTradeOpen(time.Unix(p.Timestamp, 0))
	}, printLog)
}

func file2db(filename string, start, end time.Time, onlyTxOpen bool) error {
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
	return pipeTo(fileReader, sqlWriter, func(p *pg.Price) bool {
		return (!onlyTxOpen || pg.IsTradeOpen(time.Unix(p.Timestamp, 0))) &&
			p.Timestamp >= start.Unix() && p.Timestamp < end.Unix()
	}, true)
}

func pipeTo(r pg.Reader, w pg.Writer, filterFunc func(*pg.Price) bool, printLog bool) error {
	length := r.Len()
	start, end := r.TimeRange()
	if printLog {
		log.Printf("[PIPETO] time range: %s -> %s", start.Format(pg.StampFmt), end.Format(pg.StampFmt))
		log.Printf("[PIPETO] %d records read", length)
	}
	success := 0
	for i := 0; i < length; i++ {
		p := new(pg.Price)
		if _, err := r.Read(p); err != nil {
			return fmt.Errorf("read price: %v", err)
		}
		if filterFunc(p) {
			if err := w.Write(p); err == nil {
				success++
			}
		}
		if printLog && i%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(i+1)/float32(length)*100)
		}
	}
	if printLog {
		fmt.Print("\r")
		log.Printf("[PIPETO] %d records written", success)
	}
	return nil
}
