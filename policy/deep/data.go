package deep

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// Generate deep training market price data
func Generate(c *cli.Context) error {
	filename := c.String(pg.FpComma(pg.OutfileFlag.Name))
	start, err := pg.ParseDate(c.String(pg.FpComma(pg.StartDateFlag.Name)))
	end, err := pg.ParseDate(c.String(pg.FpComma(pg.EndDateFlag.Name)))
	if err != nil {
		return fmt.Errorf("dpgen: wrong input date format: %v", err)
	}
	log.Printf("[DPGEN] db(%s) -> file(%s)", pg.DBSTR, filename)
	if err := mkTrainFile(filename, start, end); err != nil {
		return fmt.Errorf("dpgen: make train file: %v", err)
	}
	return nil
}

func mkTrainFile(filename string, start, end time.Time) (err error) {
	sqlReader, err := pg.NewSQLReader(start, end)
	if err != nil {
		return fmt.Errorf("new sql reader: %v", err)
	}
	defer sqlReader.Close()
	length := sqlReader.Len()
	start, end = sqlReader.TimeRange()
	log.Printf("[DPGEN] time range: %s -> %s", start.Format(pg.StampFmt), end.Format(pg.StampFmt))
	log.Printf("[DPGEN] %d records to read", length)
	fh, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file '%s': %v", filename, err)
	}
	defer fh.Close()
	raw := make([]pg.Price, length)
	if err := sqlReader.ReadAll(raw); err != nil {
		return fmt.Errorf("read all raw data: %v", err)
	}

	trainSet := make([]sample, 0)
	interval := time.Duration(pg.Config.Policy.DeepRateIntervalMin) * time.Minute
	deadline := end.Add(-1 * interval).Unix()
	for i, p := range raw {
		var single sample
		tm := time.Unix(p.Timestamp, 0)
		if i == 0 || p.Timestamp > deadline || !pg.IsTradeOpen(tm) {
			continue
		}
		single.Timestamp = p.Timestamp
		ratio := float32(tm.Sub(time.Unix(raw[i-1].Timestamp, 0))) / float32((time.Duration(pg.Config.DB.TickSec) * time.Second))
		single.PriceDiff = (p.Bankbuy - raw[i-1].Bankbuy) / ratio * 5

		var avg float32
		j := i + 1
		m := time.Unix(p.Timestamp, 0).Add(interval).Unix()
		for j < length && raw[j].Timestamp <= m {
			avg += raw[j].Bankbuy
			j++
		}
		avg /= float32(j - i - 1)
		switch {
		case avg-p.Bankbuy >= pg.Config.Policy.DeepRateBuyLimit:
			single.Action = ShouldBuy
		case p.Bankbuy-avg >= pg.Config.Policy.DeepRateSellLimit:
			single.Action = ShouldSell
		default:
			single.Action = ShouldHold
		}

		trainSet = append(trainSet, single)

		if i%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(i+1)/float32(length)*100)
		}
	}

	if err := binary.Write(fh, binary.LittleEndian, int32(len(trainSet))); err != nil {
		return fmt.Errorf("write length header: %v", err)
	}
	if err := binary.Write(fh, binary.LittleEndian, trainSet); err != nil {
		return fmt.Errorf("write train sample set: %v", err)
	}

	fmt.Print("\r")
	log.Printf("[DPGEN] %d records written", len(trainSet))

	return nil
}

type sample struct {
	Timestamp int64
	PriceDiff float32
	Action    uint8
}

// Action should taken
const (
	ShouldBuy uint8 = 1 << iota
	ShouldSell
	ShouldHold
)

func (s sample) String() string {
	tm := time.Unix(s.Timestamp, 0)
	action := ""
	switch s.Action {
	case ShouldBuy:
		action = "buy"
	case ShouldSell:
		action = "sell"
	case ShouldHold:
		action = " "
	default:
		panic("inconsistent action type")
	}
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d | %5.2f | %s",
		tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), s.PriceDiff, action)
}
