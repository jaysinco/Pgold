package server

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// Run pgold server
func Run(c *cli.Context) error {
	log.Println("[SERVER] run")

	baseDir := filepath.ToSlash(pg.Config.Server.Basedir)
	if baseDir == "default" || baseDir == "" {
		baseDir = pg.SourceDir + "/server/public"
	}
	log.Printf("[SERVER] base directory is '%s'", baseDir)

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	pmset, err := getPoemSet(baseDir + "/text/poem.txt")
	if err != nil {
		return fmt.Errorf("server: prepare poem set: %v", err)
	}
	log.Printf("[SERVER] %d poems loaded\n", len(pmset))

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			http.Redirect(w, r, "/public/favicon.ico", http.StatusMovedPermanently)
			return
		}
		http.Redirect(w, r, "/public/html/home.html", http.StatusMovedPermanently)
	}))
	mux.Handle("/public/", http.StripPrefix("/public", http.FileServer(http.Dir(baseDir))))
	mux.Handle("/papergold/price/tick/json/by/timestamp", new(tickPrice))
	mux.Handle("/papergold/price/kline/json/all/day", new(klinePrice))
	mux.Handle("/poem/random", &randomPoet{Rnd: rnd, Set: pmset})

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(pg.Config.Server.Port),
		Handler: mux,
	}
	log.Printf("[SERVER] listening on port%s\n", server.Addr)
	return fmt.Errorf("server: stop unexpectedly: %v", server.ListenAndServe())
}

type klinePrice struct{}

func (kp *klinePrice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pgks, err := readDaykPrice(pg.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("query day kline price: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.Encode(pgks)
}

type pgkday struct {
	Timestamp int64   `json:"t"`
	Open      float32 `json:"o"`
	High      float32 `json:"h"`
	Low       float32 `json:"l"`
	Close     float32 `json:"c"`
}

func (p pgkday) String() string {
	tm := time.Unix(p.Timestamp, 0)
	return fmt.Sprintf("%4d-%02d-%02d | open: %.2f/ high: %.2f/ low: %.2f/ close: %.2f",
		tm.Year(), tm.Month(), tm.Day(), p.Open, p.High, p.Low, p.Close)
}

func readDaykPrice(db *sql.DB) ([]pgkday, error) {
	rows, err := db.Query(`
		SELECT to_timestamp(date, 'YYYY-MM-DD') daytmsp,
			(SELECT bankbuy FROM pgmkt WHERE txtime = min(tmp.txtime)) open,
			max(bankbuy) high, 
			min(bankbuy) low,
			(SELECT bankbuy FROM pgmkt WHERE txtime = max(tmp.txtime)) closep
		FROM (SELECT to_char(txtime, 'YYYY-MM-DD') date, txtime, bankbuy FROM pgmkt ORDER BY txtime) tmp
		GROUP BY date ORDER BY date`)
	if err != nil {
		return nil, fmt.Errorf("select kline from 'pgmkt': %v", err)
	}
	defer rows.Close()
	tm := new(time.Time)
	pgks := make([]pgkday, 0)
	for rows.Next() {
		var pgd pgkday
		if err := rows.Scan(tm, &pgd.Open, &pgd.High, &pgd.Low, &pgd.Close); err != nil {
			return nil, fmt.Errorf("scan rows: %v", err)
		}
		pgd.Timestamp = tm.Unix()
		if pgd.High != pgd.Low {
			pgks = append(pgks, pgd)
		}
	}
	return pgks, nil
}

type tickPrice struct{}

func (tp *tickPrice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	start, err1 := strconv.ParseInt(r.Form.Get("start"), 10, 64)
	end, err2 := strconv.ParseInt(r.Form.Get("end"), 10, 64)
	if err1 != nil || err2 != nil {
		http.Error(w, fmt.Sprintf("request timestamp parameter: start=%v(%v); end=%v(%v)",
			start, err1, end, err2), http.StatusBadRequest)
		return
	}
	bts, err := readTickBuy(pg.DB, start, end)
	if err != nil {
		http.Error(w, fmt.Sprintf("query papergold tick data: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.Encode(bts)
}

type pgtbuy struct {
	Timestamp int64   `json:"t"`
	Bankbuy   float32 `json:"p"`
}

func readTickBuy(db *sql.DB, start, end int64) ([]pgtbuy, error) {
	rows, err := db.Query(`
		SELECT txtime, bankbuy
		FROM pgmkt
		WHERE extract(epoch from txtime) >= $1 and extract(epoch from txtime) <= $2 order by txtime`, start, end)
	if err != nil {
		return nil, fmt.Errorf("select buy from 'pgmkt': %v", err)
	}
	defer rows.Close()
	tm := new(time.Time)
	pgts := make([]pgtbuy, 0)
	for rows.Next() {
		var pgt pgtbuy
		if err := rows.Scan(tm, &pgt.Bankbuy); err != nil {
			return nil, fmt.Errorf("scan rows: %v", err)
		}
		pgt.Timestamp = tm.Unix()
		pgts = append(pgts, pgt)
	}
	return pgts, nil
}

type randomPoet struct {
	Rnd *rand.Rand
	Set []*poetry
}

func (rp *randomPoet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pm := *rp.Set[rp.Rnd.Intn(len(rp.Set))]
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.Encode(pm)
}

func getPoemSet(filename string) ([]*poetry, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file %v: %v", filename, err)
	}
	defer fp.Close()
	set := make([]*poetry, 0)
	reader := bufio.NewReader(fp)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		parts := strings.Split(string(line), "::")
		if len(parts) != 3 {
			continue
		}
		pm := &poetry{parts[0], parts[1], strings.Split(parts[2], "/")}
		if checkLength(pm.Paragraphs) {
			set = append(set, pm)
		}
	}

	return set, nil
}

func checkLength(para []string) bool {
	if len(para) == 0 {
		return false
	}
	max, min := 0, 999999
	for _, s := range para {
		c := utf8.RuneCountInString(s)
		if c > max {
			max = c
		}
		if c < min {
			min = c
		}
	}
	if max > 16 || min < 8 {
		return false
	}
	return true
}

type poetry struct {
	Title      string
	Author     string
	Paragraphs []string
}
