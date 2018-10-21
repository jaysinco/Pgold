package show

import (
	"bufio"
	"database/sql"
	"encoding/binary"
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

	"github.com/jaysinco/Pgold/utils"
	"github.com/urfave/cli"
)

// ShowCmd run show subcommand
var ShowCmd = cli.Command{
	Name:   "show",
	Usage:  "show market history data through http server",
	Action: showRun,
}

func showRun(c *cli.Context) error {
	log.Println("start market showing server")

	config, err := utils.LoadConfigFile(c.GlobalString(utils.ConfigFlag.Name))
	if err != nil {
		log.Fatalf("load configure file: %v", err)
	}

	db, err := utils.SetupDatabase(&config.DB)
	if err != nil {
		log.Fatalf("setup database: %v", err)
	}
	defer db.Close()

	baseDir := filepath.ToSlash(config.Show.Base)
	if baseDir == "default" || baseDir == "" {
		baseDir = filepath.ToSlash(os.Getenv("GOPATH")) + "/src/github.com/jaysinco/Pgold/show/public"
	}
	log.Printf("base directory is '%s'", baseDir)

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	pmset, err := mkPoemList(baseDir + "/text/poem.txt")
	if err != nil {
		log.Fatalf("prepare poem set: %v\n", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			http.Redirect(w, r, "/public/favicon.ico", http.StatusMovedPermanently)
			return
		}
		http.Redirect(w, r, "/public/html/home.html", http.StatusMovedPermanently)
	}))
	mux.Handle("/public/", http.StripPrefix("/public", http.FileServer(http.Dir(baseDir))))
	mux.Handle("/papergold/price/tick/json/by/timestamp", &tickPrice{DB: db, Mode: typeJSON})
	mux.Handle("/papergold/price/kline/json/all/day", &klinePrice{DB: db})
	mux.Handle("/poem/random", &randomPoet{Rnd: rnd, Set: pmset})

	server := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}
	log.Printf("listening on port%s\n", server.Addr)
	log.Printf("stop unexpectedly: %v\n", server.ListenAndServe())
	return nil
}

type klinePrice struct {
	DB *sql.DB
}

func (kp *klinePrice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pgks, err := queryKLineData(kp.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("query papergold kline data: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.Encode(pgks)
}

type pgkday struct {
	Timestamp float64 `json:"t"`
	Open      float32 `json:"o"`
	High      float32 `json:"h"`
	Low       float32 `json:"l"`
	Close     float32 `json:"c"`
}

func (p pgkday) String() string {
	tm := time.Unix(int64(p.Timestamp), 0)
	return fmt.Sprintf("%4d-%02d-%02d | open: %.2f/ high: %.2f/ low: %.2f/ close: %.2f",
		tm.Year(), tm.Month(), tm.Day(), p.Open, p.High, p.Low, p.Close)
}

func queryKLineData(db *sql.DB) ([]pgkday, error) {
	rows, err := db.Query(`
		SELECT extract(epoch from to_timestamp(date, 'YYYY-MM-DD')) daytmsp,
			(SELECT bankbuy FROM pgmkt WHERE txtime = min(tmp.txtime)) open,
			max(bankbuy) high, 
			min(bankbuy) low,
			(SELECT bankbuy FROM pgmkt WHERE txtime = max(tmp.txtime)) closep
		FROM (SELECT to_char(txtime, 'YYYY-MM-DD') date, txtime, bankbuy FROM pgmkt ORDER BY txtime) tmp
		GROUP BY date ORDER BY date`)
	if err != nil {
		return nil, fmt.Errorf("query data from table 'pgmkt': %v", err)
	}
	defer rows.Close()
	pgks := make([]pgkday, 0)
	for rows.Next() {
		var pgd pgkday
		if err := rows.Scan(&pgd.Timestamp, &pgd.Open, &pgd.High, &pgd.Low, &pgd.Close); err != nil {
			return nil, fmt.Errorf("scan rows: %v", err)
		}
		if pgd.High != pgd.Low {
			pgks = append(pgks, pgd)
		}
	}
	return pgks, nil
}

type tickPrice struct {
	DB   *sql.DB
	Mode respFile
}

func (tp *tickPrice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	start, err1 := strconv.ParseInt(r.Form.Get("start"), 10, 64)
	end, err2 := strconv.ParseInt(r.Form.Get("end"), 10, 64)
	if err1 != nil || err2 != nil {
		http.Error(w, fmt.Sprintf("request timestamp parameter: start=%v(%v); end=%v(%v)",
			start, err1, end, err2), http.StatusBadRequest)
		return
	}
	pgcs, err := queryTickData(tp.DB, start, end)
	if err != nil {
		http.Error(w, fmt.Sprintf("query papergold tick data: %v", err), http.StatusInternalServerError)
		return
	}
	switch tp.Mode {
	case typeBinary:
		w.Header().Set("Content-Type", "application/octet-stream")
		writeBinaryTick(pgcs, w)
	case typeJSON:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		writeJSONTick(pgcs, w)
	}
}

type respFile int

const (
	typeJSON respFile = iota
	typeBinary
)

type pgprice struct {
	Timestamp float64 `json:"t"`
	Bankbuy   float32 `json:"p"`
}

func (p pgprice) String() string {
	tm := time.Unix(int64(p.Timestamp), 0)
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d | %.2f",
		tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), p.Bankbuy)
}

func queryTickData(db *sql.DB, start, end int64) ([]pgprice, error) {
	rows, err := db.Query(`
		SELECT txtmsp, bankbuy
		FROM (SELECT extract(epoch from txtime) txtmsp, bankbuy FROM pgmkt ORDER BY txtime) tmp
		WHERE txtmsp >= $1 and txtmsp <= $2`, start, end)
	if err != nil {
		return nil, fmt.Errorf("query data from table 'pgmkt': %v", err)
	}
	defer rows.Close()
	pgcs := make([]pgprice, 0)
	for rows.Next() {
		var pgc pgprice
		if err := rows.Scan(&pgc.Timestamp, &pgc.Bankbuy); err != nil {
			return nil, fmt.Errorf("scan rows: %v", err)
		}
		pgcs = append(pgcs, pgc)
	}
	return pgcs, nil
}

func writeJSONTick(pgcs []pgprice, out io.Writer) error {
	encoder := json.NewEncoder(out)
	return encoder.Encode(pgcs)
}

func writeBinaryTick(pgcs []pgprice, out io.Writer) error {
	num := len(pgcs)
	if err := binary.Write(out, binary.BigEndian, int32(num)); err != nil {
		return fmt.Errorf("write record num header: %v", err)
	}
	if err := binary.Write(out, binary.BigEndian, pgcs); err != nil {
		return fmt.Errorf("write records: %v", err)
	}
	return nil
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

func mkPoemList(filename string) ([]*poetry, error) {
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
		set = append(set, pm)
	}
	return set, nil
}

type poetry struct {
	Title      string
	Author     string
	Paragraphs []string
}
