package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	host := flag.String("h", "127.0.0.1", "host for database configure")
	user := flag.String("U", "root", "user for database configure")
	dbname := flag.String("d", "root", "dbname for database configure")
	passwd := flag.String("p", "unknown", "passwords for database configure")
	flag.Parse()

	config := fmt.Sprintf("host=%s password=%s user=%s dbname=%s sslmode=disable",
		*host, *passwd, *user, *dbname)
	db, err := sql.Open("postgres", config)
	if err != nil {
		log.Fatalf("[ERROR] open postgres[%s]: %v", config, err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[ERROR] connect to postgres[%s]: %v", config, err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/wx", logRequest(procWechat, db))
	mux.HandleFunc("/pg/plot", logRequest(procPaperGold, db))
	server := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}
	log.Printf("jserver is listening on port%s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[ERROR] jserver stop: %v", err)
	}
}

type StdLogger func(format string, values ...interface{})

type HandlerFuncWithMore func(w http.ResponseWriter, r *http.Request, mlog StdLogger, db *sql.DB)

func logRequest(hdl HandlerFuncWithMore, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mlog := func(format string, values ...interface{}) {
			log.Printf("[%s] %s", r.RemoteAddr, fmt.Sprintf(format, values...))
		}
		mlog(`- "%s %s"`, r.Method, r.URL)
		if err := r.ParseForm(); err != nil {
			mlog("[ERROR] parse form: %v", err)
		}
		hdl(w, r, mlog, db)
	}
}
