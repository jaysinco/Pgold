package policy

import (
	"fmt"
	"log"
	"time"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// Test strategy based on history price data
func Test(c *cli.Context) error {
	start, err := pg.ParseDate(c.String(pg.FpComma(pg.StartDateFlag.Name)))
	end, err := pg.ParseDate(c.String(pg.FpComma(pg.EndDateFlag.Name)))
	if err != nil {
		return fmt.Errorf("test: wrong input date format: %v", err)
	}
	sqlReader, err := pg.NewSQLReader(start, end)
	if err != nil {
		return fmt.Errorf("test: new sql reader: %v", err)
	}
	defer sqlReader.Close()
	start, end = sqlReader.TimeRange()

	chosen := c.String(pg.FpComma(pg.PolicyFlag.Name))
	var stra strategy
	for _, p := range globalPolicySet {
		if p.Name == chosen {
			stra = p.CreateMethod(start, end)
			break
		}
	}
	if stra == nil {
		return fmt.Errorf("test: policy name not registered: '%s'", chosen)
	}
	log.Printf("[TEST] policy name : %s", chosen)

	return loopbackTest(stra, sqlReader)
}

func loopbackTest(s strategy, r pg.Reader) error {
	length := r.Len()
	start, end := r.TimeRange()
	log.Printf("[TEST] time range  : %s -> %s", start.Format(pg.StampFmt), end.Format(pg.StampFmt))
	log.Printf("[TEST] total ticks : %d", length)
	log.Printf("[TEST] ************** START **************")

	txb := txBook{start, end, make([]*txRecord, 0)}
	var p pg.Price
	ctx := tradeContex{&p}
	for i := 0; i < length; i++ {
		if _, err := r.Read(&p); err != nil {
			return fmt.Errorf("read price: %v", err)
		}
		if pg.IsTradeOpen(time.Unix(p.Timestamp, 0)) {
			sig, msg := s.Dealwith(&ctx)
			switch sig {
			case Pass:
				continue
			case Buy:
				rec := &txRecord{'B', time.Unix(p.Timestamp, 0), p.Banksell, msg}
				txb.Rec = append(txb.Rec, rec)
				log.Printf("[TEST] %s\n", rec)
			case Sell:
				rec := &txRecord{'S', time.Unix(p.Timestamp, 0), p.Bankbuy, msg}
				txb.Rec = append(txb.Rec, rec)
				log.Printf("[TEST] %s\n", rec)
			case Warn:
				log.Printf("[TEST] [W] %s %s\n", time.Unix(p.Timestamp, 0).Format(pg.StampFmt), msg)
			}
		}
		if i%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(i+1)/float32(length)*100)
		}
		fmt.Printf("\r")
	}
	log.Printf("[TEST] *********** END OF TEST ***********")
	log.Printf("[TEST] ** annual rate of return : %.2f%%", txb.AnnualReturn()*100)
	log.Printf("[TEST] ** trading frequency     : %.1f/day", txb.TradeFreqPerDay())
	return nil
}

type txRecord struct {
	Action rune
	Time   time.Time
	Amount float32
	Remark string
}

func (tr *txRecord) String() string {
	var ind rune
	if tr.Action == 'B' {
		ind = '-'
	} else {
		ind = '+'
	}
	return fmt.Sprintf("[%c] %s $%c%.2f %s", tr.Action,
		tr.Time.Format(pg.StampFmt), ind, tr.Amount, tr.Remark)
}

type txBook struct {
	Start time.Time
	End   time.Time
	Rec   []*txRecord
}

func (tb *txBook) AnnualReturn() float32 {
	var balance float32
	var cost float32
	sub := tb.End.Sub(tb.Start)
	for i, r := range tb.Rec {
		if i == len(tb.Rec)-1 && r.Action == 'B' {
			break
		}
		switch r.Action {
		case 'B':
			balance -= r.Amount
			cost += r.Amount
		case 'S':
			balance += r.Amount
		default:
			panic("should not reach here!")
		}
	}
	if cost == 0 {
		return 0
	}
	return balance / cost / float32(sub.Hours()/24) * 365
}

func (tb *txBook) TradeFreqPerDay() float64 {
	return float64(len(tb.Rec)) / (tb.End.Sub(tb.Start).Hours() / 24)
}
