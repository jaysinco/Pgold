package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/golang/glog"
	_ "github.com/lib/pq"
)

func main() {
	host := flag.String("h", "106.15.194.234", "host for database configure")
	user := flag.String("u", "root", "user for database configure")
	dbname := flag.String("d", "root", "dbname for database configure")
	passwd := flag.String("p", "unknown", "passwords for database configure")
	sslmode := flag.String("s", "disable", "sslmode for database configure")
	flag.Parse()

	config := fmt.Sprintf("host=%s password=%s user=%s dbname=%s sslmode=%s",
		*host, *passwd, *user, *dbname, *sslmode)
	db, _ := sql.Open("postgres", config)
	if err := db.Ping(); err != nil {
		glog.V(0).Infof("connect to postgres[%s]: %v", config, err)
		return
	}
	defer db.Close()
	tbname := "pgmkt"
	if err := createMktTbl(db, tbname); err != nil {
		glog.V(0).Infof("create table %s: %v", tbname, err)
		return
	}

	tick := 30 * time.Second
	wait := 5 * time.Second
	for {
		epochBegin := time.Now()
		for {
			if err := insertMktData(db, tbname); err != nil {
				glog.V(0).Infof("insert market data: %v", err)
				if time.Since(epochBegin)+wait > tick {
					glog.V(0).Infof("*** quit due to timeout ***")
					return
				}
				time.Sleep(wait)
			} else {
				time.Sleep(tick - time.Since(epochBegin))
				break
			}
		}
	}
}

func insertMktData(db *sql.DB, tbname string) error {
	buy, sell, err := queryPaperGold()
	if err != nil {
		return fmt.Errorf("query paper gold: %v", err)
	}
	_, err = db.Exec(fmt.Sprintf(`insert into %s(txtime,bankbuy,banksell) values('now',%.2f,%.2f)`,
		tbname, buy, sell))
	return err
}

func createMktTbl(db *sql.DB, tbname string) error {
	_, err := db.Exec(fmt.Sprintf(`create table if not exists %s(
		txtime    timestamp(0) without time zone primary key,
		bankbuy   numeric(8,2),
		banksell  numeric(8,2)
	)`, tbname))
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
		err = fmt.Errorf("price pattern match failed, body: %s", string(body))
		return
	}
	bankBuyPrice, _ = strconv.ParseFloat(string(prices[1]), 64)
	bankSellPrice, _ = strconv.ParseFloat(string(prices[2]), 64)
	return
}
