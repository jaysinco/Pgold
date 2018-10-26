package hint

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jaysinco/Pgold/utils"
	"github.com/urfave/cli"
)

func hintRun(c *cli.Context) error {
	log.Println("start sending trade tips")

	tick := 30 * time.Second
	for {
		if err := checkWarning(utils.DB, utils.Config); err != nil {
			log.Printf("check warning: %v", err)
		}
		time.Sleep(tick)
	}
}

func checkWarning(db *sql.DB, config *utils.TomlConfig) error {
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
		log.Printf("upper threshold reached(%.2f-%.2f>%.2f) during last %s",
			nowVal, minVal, threshold, duration)
		sub = fmt.Sprintf("Paper gold is up %.2f RMB/g during last %d minutes.",
			nowVal-minVal, duration/time.Minute)
		body = "(๑•̀ㅂ•́)و✧"
	}
	if maxVal-nowVal > threshold {
		log.Printf("lower threshold reached(%.2f-%.2f>%.2f) during last %s",
			maxVal, nowVal, threshold, duration)
		sub = fmt.Sprintf("Paper gold is down %.2f RMB/g during last %d minutes.",
			maxVal-nowVal, duration/time.Minute)
		body = "థ౪థ.........."
	}

	if sub != "" || body != "" {
		if err := utils.SendMail(sub, body, &config.Mail); err != nil {
			return fmt.Errorf("send email: %v", err)
		}
		time.Sleep(duration)
	}
	return nil
}
