package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	host := flag.String("h", "127.0.0.1", "server host of postgreSQL")
	user := flag.String("U", "root", "user name of postgreSQL")
	dbname := flag.String("d", "root", "database name of postgreSQL")
	passwd := flag.String("p", "unknown", "login password of postgreSQL")
	flag.Parse()

	log.Println("[PGBAN] start")
	token := fmt.Sprintf("host=%s password=%s user=%s dbname=%s sslmode=disable",
		*host, *passwd, *user, *dbname)
	db, err := sql.Open("postgres", token)
	if err != nil {
		log.Fatalf("[PGBAN] open postgres[%s]: %v", token, err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[PGBAN] connect to postgres[%s]: %v", token, err)
	}
	defer db.Close()

	tick := 30 * time.Second
	for {
		if err := checkWarning(db); err != nil {
			log.Printf("[PGBAN] check warning: %v", err)
		}
		time.Sleep(tick)
	}
}

func checkWarning(db *sql.DB) error {
	threshold := float32(0.75)
	duration := 45 * time.Minute

	start := time.Now().Add(-1 * duration).Unix()
	row, err := db.Query(`
		SELECT MAX(bankbuy) maxbuy, 
			   MIN(bankbuy) minbuy, 
			   (SELECT bankbuy FROM pgmkt WHERE extract(epoch from txtime) = MAX(txtmsp)) nowbuy
		FROM (SELECT extract(epoch from txtime) txtmsp, bankbuy FROM pgmkt ORDER BY txtime) tmp
		WHERE txtmsp >= $1`, start)
	if err != nil {
		return fmt.Errorf("query data from table 'pgmkt': %v", err)
	}
	defer row.Close()
	if ok := row.Next(); !ok {
		return fmt.Errorf("next row empty: haven't received any data from pgmkt since last %v", duration)
	}
	var maxVal, minVal, nowVal float32
	if err := row.Scan(&maxVal, &minVal, &nowVal); err != nil {
		return fmt.Errorf("scan rows: %v", err)
	}

	var sub, body string
	if nowVal-minVal > threshold {
		log.Printf("[PGBAN] upper threshold reached(%.2f-%.2f>%.2f) during last %s",
			nowVal, minVal, threshold, duration)
		sub = fmt.Sprintf("Paper gold is up %.2f RMB/g during last %d minutes.",
			nowVal-minVal, duration/time.Minute)
		body = "(๑•̀ㅂ•́)و✧"
	}
	if maxVal-nowVal > threshold {
		log.Printf("[PGBAN] lower threshold reached(%.2f-%.2f>%.2f) during last %s",
			maxVal, nowVal, threshold, duration)
		sub = fmt.Sprintf("Paper gold is down %.2f RMB/g during last %d minutes.",
			maxVal-nowVal, duration/time.Minute)
		body = "థ౪థ.........."
	}

	if sub != "" || body != "" {
		if err := sendMail(sub, body); err != nil {
			return fmt.Errorf("send email: %v", err)
		}
		time.Sleep(duration)
	}
	return nil
}

func sendMail(subject, body string) error {
	from := "jaysinco@qq.com"
	to := "jaysinco@163.com;1052386099@qq.com;tracytangshi@163.com"
	pwd := "oolmgpqhbvqyicfb"
	domain := from[strings.Index(from, "@")+1:]
	auth := smtp.PlainAuth("", from, pwd, fmt.Sprintf("smtp.%s", domain))
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Subject: %s\r\n"+
		"\r\n%s\r\n", from, to, subject, body)
	return smtp.SendMail(fmt.Sprintf("smtp.%s:25", domain), auth,
		from, strings.Split(to, ";"), []byte(msg))
}
