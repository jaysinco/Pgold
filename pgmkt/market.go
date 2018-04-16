package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	host := flag.String("h", "127.0.0.1", "host for database configure")
	user := flag.String("U", "root", "user for database configure")
	dbname := flag.String("d", "root", "dbname for database configure")
	passwd := flag.String("p", "unknown", "passwords for database configure")
	flag.Parse()

	log.Println("[PGMKT] start")
	config := fmt.Sprintf("host=%s password=%s user=%s dbname=%s sslmode=disable",
		*host, *passwd, *user, *dbname)
	db, err := sql.Open("postgres", config)
	if err != nil {
		log.Fatalf("[PGMKT] open postgres[%s]: %v", config, err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[PGMKT] connect to postgres[%s]: %v", config, err)
	}
	defer db.Close()
	if err := createMktTbl(db); err != nil {
		log.Fatalf("[PGMKT] create market table 'pgmkt': %v", err)
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
					log.Println("[PGMKT] fixing error encountered when update market data")
					retry = true
				}
				ecount[err.Error()]++
				time.Sleep(wait)
			} else {
				if retry {
					var report bytes.Buffer
					for ers, t := range ecount {
						if len(ers) > 80 {
							ers = ers[:80] + "..."
						}
						report.WriteString(fmt.Sprintf("\n - %s(%d times)", ers, t))
					}
					log.Printf("[PGMKT] problem solved, error review:%s", report.String())
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
