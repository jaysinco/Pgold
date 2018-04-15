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
	pwd := flag.String("p", "unknown", "login password of postgreSQL")
	flag.Parse()

	log.Println("*** PAPER GOLD BRAIN ***")
	token := fmt.Sprintf("host=%s password=%s user=root dbname=root sslmode=disable", *host, *pwd)
	db, err := sql.Open("postgres", token)
	if err != nil {
		log.Fatalf("open postgres[%s]: %v", token, err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("connect to postgres[%s]: %v", token, err)
	}
	defer db.Close()

	var lstWarnTm time.Time
	tick := 30 * time.Second
	warnSep := 60 * time.Minute
	for {
		if time.Since(lstWarnTm) > warnSep {
			warning, err := checkWarning(db)
			if err != nil {
				log.Printf("check warning: %v", err)
			}
			if warning != "" {
				body := fmt.Sprintf(`<ul style='font-size:15px'>%s</ul>`, warning)
				if err := sendMail("纸黄金价格波动提示", body); err != nil {
					log.Printf("send email: %v", err)
				} else {
					lstWarnTm = time.Now()
				}
			}
		}
		time.Sleep(tick)
	}
}

func checkWarning(db *sql.DB) (string, error) {
	var warning string
	// -warning: [peak] check
	threshold := float32(1.0)
	duration := 30 * time.Minute
	start := time.Now().Add(-1 * duration).Unix()
	row, err := db.Query(`
		SELECT MAX(bankbuy) maxbuy, 
			   MIN(bankbuy) minbuy, 
			   (SELECT bankbuy FROM pgmkt WHERE extract(epoch from txtime) = MAX(txtmsp)) nowbuy
		FROM (SELECT extract(epoch from txtime) txtmsp, bankbuy FROM pgmkt ORDER BY txtime) tmp
		WHERE txtmsp >= $1`, start)
	if err != nil {
		return warning, fmt.Errorf("query data from table 'pgmkt': %v", err)
	}
	defer row.Close()
	row.Next()
	var maxVal, minVal, nowVal float32
	if err := row.Scan(&maxVal, &minVal, &nowVal); err != nil {
		return warning, fmt.Errorf("scan rows: %v", err)
	}
	if (maxVal-nowVal > threshold) || (nowVal-minVal > threshold) {
		warning += fmt.Sprintf(`<li>threshold reached since last <u>%s</u> 
        	<br>—— [R]<b>%.2f</b> / [H]<b>%.2f</b> / [L]<b>%.2f</b></br>
    		</li>`, duration, nowVal, maxVal, minVal)
	}
	return warning, nil
}

func sendMail(subject, body string) error {
	from := "jaysinco@qq.com"
	to := "jaysinco@163.com"
	pwd := "ygkstvxfsovkific"
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
