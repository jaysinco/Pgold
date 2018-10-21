package market

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jaysinco/Pgold/utils"
	"github.com/urfave/cli"
)

// MarketCmd run market subcommand
var MarketCmd = cli.Command{
	Name:   "market",
	Usage:  "fetch market data into database continuously",
	Action: marketRun,
}

func marketRun(c *cli.Context) error {
	log.Println("start fetching market data")

	config, err := utils.LoadConfigFile(c.GlobalString(utils.ConfigFlag.Name))
	if err != nil {
		log.Fatalf("load configure file: %v", err)
	}

	db, err := utils.SetupDatabase(&config.DB)
	if err != nil {
		log.Fatalf("setup database: %v", err)
	}
	defer db.Close()

	if err := createMktTbl(db); err != nil {
		log.Fatalf("create market table 'pgmkt': %v", err)
	}

	tick := 30 * time.Second
	wait := 5 * time.Second
	for {
		retry := false
		ecount := make(map[string]int)
		epochBegin := time.Now()
		for {
			if err := insertMktData(db); err != nil {
				if !retry {
					retry = true
				}
				ecount[err.Error()]++
				time.Sleep(wait)
			} else {
				if retry {
					var report bytes.Buffer
					for ers, t := range ecount {
						if len(ers) > 100 {
							ers = ers[:100] + "..."
						}
						ers = strings.Replace(ers, "\n", "", -1)
						report.WriteString(fmt.Sprintf("%s(%d times);", ers, t))
					}
					log.Printf("error encountered then fixed => %s", report.String())
				}
				stm := tick - time.Since(epochBegin)
				if stm < wait {
					stm = wait
				}
				time.Sleep(stm)
				break
			}
		}
	}
}

func insertMktData(db *sql.DB) error {
	buy, sell, err := queryPaperGold()
	if err != nil {
		return fmt.Errorf("query paper gold: %v", err)
	}
	_, err = db.Exec("insert into pgmkt(txtime,bankbuy,banksell) values('now',$1,$2)", buy, sell)
	return err
}

func createMktTbl(db *sql.DB) error {
	_, err := db.Exec(`create table if not exists pgmkt(
		txtime    timestamp(0) with time zone primary key,
		bankbuy   numeric(8,2),
		banksell  numeric(8,2)
	)`)
	return err
}

var pricePatt = regexp.MustCompile(`人民币账户黄金(?s:.)*?(\d\d\d\.\d\d)(?s:.)*?(\d\d\d\.\d\d)`)

func queryPaperGold() (bankBuyPrice, bankSellPrice float64, err error) {
	resp, err := http.Get("http://www.icbc.com.cn/ICBCDynamicSite/Charts/GoldTendencyPicture.aspx")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("read body: %v", err)
		return
	}
	prices := pricePatt.FindSubmatch(body)
	if len(prices) != 3 {
		err = fmt.Errorf("price pattern match failed within body: %s", string(body))
		return
	}
	bankBuyPrice, _ = strconv.ParseFloat(string(prices[1]), 64)
	bankSellPrice, _ = strconv.ParseFloat(string(prices[2]), 64)
	return
}
