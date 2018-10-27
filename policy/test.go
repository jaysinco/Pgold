package hint

import (
	"fmt"
	"log"
	"time"

	"github.com/urfave/cli"
)

func testRun(c *cli.Context) error {
	start, err := utils.ParseDate(c.String(utils.GetFlagName(utils.StartDateFlag)))
	end, err := utils.ParseDate(c.String(utils.GetFlagName(utils.EndDateFlag)))
	if err != nil {
		return fmt.Errorf("wrong input date format: %v", err)
	}
	source, err := newDBSource(start, end, utils.DB)
	if err != nil {
		return fmt.Errorf("create data iterator: %v", err)
	}
	defer source.Close()

	stra := newRandTester(13)
	loopbackTest(stra, source)

	return nil
}

func loopbackTest(stra strategy, source dataIter) {
	log.Printf("strategy name: '%s'", stra.Name())

	start, end := source.TimeRange()
	log.Printf("test range: %s -> % s",
		start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))
	total := source.Len()
	log.Printf("test scale: %d ticks", total)
	log.Println("************* START *************")

	txb := txBook{start, end, make([]*txRecord, 0)}
	for {
		data, dry := source.Next()
		if dry {
			break
		}
		ctx := &tradeContex{data, utils.DB}
		sig, msg := stra.Dealwith(ctx)
		switch sig {
		case Pass:
			continue
		case Buy:
			rec := &txRecord{'B', data.Txtime, data.Banksell, msg}
			txb.Rec = append(txb.Rec, rec)
			log.Println(rec)
		case Sell:
			rec := &txRecord{'S', data.Txtime, data.Bankbuy, msg}
			txb.Rec = append(txb.Rec, rec)
			log.Println(rec)
		case Warn:
			log.Println(msg)
		}
	}
	log.Printf("********** END OF TEST **********")
	log.Printf("** annual rate of return: %.3f%%", txb.AnnualReturn()*100)
	log.Printf("** trading frequency : %.2f/day", txb.TradeFreq())
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
		tr.Time.Format("2006-01-02 15:04:05"), ind, tr.Amount, tr.Remark)
}

type txBook struct {
	Start time.Time
	End   time.Time
	Rec   []*txRecord
}

func (tb *txBook) AnnualReturn() float32 {
	return 0
}

func (tb *txBook) TradeFreq() float32 {
	return 0
}
