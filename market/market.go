package market

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// Run market data update continuously
func Run(c *cli.Context) error {
	log.Println("[MARKET] run")
	sqlWriter := pg.NewSQLWriter()
	if err := pg.CreateMktTbl(); err != nil {
		return fmt.Errorf("market: %v", err)
	}
	tick := 30 * time.Second
	wait := 5 * time.Second
	for {
		retry := false
		ecount := make(map[string]int)
		epochBegin := time.Now()
		for {
			if err := updateMarket(sqlWriter); err != nil {
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

func updateMarket(w pg.Writer) error {
	p, err := crawlPrice()
	if err == nil {
		return fmt.Errorf("crawl price: %v", err)
	}
	return w.Write(p)
}

var pgMatcher = regexp.MustCompile(`人民币账户黄金(?s:.)*?(\d\d\d\.\d\d)(?s:.)*?(\d\d\d\.\d\d)`)

func crawlPrice() (*pg.Price, error) {
	resp, err := http.Get("http://www.icbc.com.cn/ICBCDynamicSite/Charts/GoldTendencyPicture.aspx")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}
	prices := pgMatcher.FindSubmatch(body)
	if len(prices) != 3 {
		return nil, fmt.Errorf("match price failed within body: %s", string(body))
	}
	buy, _ := strconv.ParseFloat(string(prices[1]), 32)
	sell, _ := strconv.ParseFloat(string(prices[2]), 32)
	p := new(pg.Price)
	p.Bankbuy, p.Banksell = float32(buy), float32(sell)
	p.Timestamp = time.Now().Unix()
	return p, nil
}
