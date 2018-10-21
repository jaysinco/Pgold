package utils

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
)

// General settings
var (
	ConfigFlag = cli.StringFlag{
		Name:  "config",
		Value: "./pgold.conf",
		Usage: "load configuration from `FILE`",
	}
	InfileFlag = cli.StringFlag{
		Name:  "in",
		Usage: "read input from `FILE`",
	}
	OutfileFlag = cli.StringFlag{
		Name:  "out",
		Usage: "write output into `FILE`",
	}
	OnlyTxOpenFlag = cli.BoolFlag{
		Name:  "tx-only",
		Usage: "only when transaction open",
	}
)

// TomlConfig stands for configure file
type TomlConfig struct {
	DB   DBInfo `toml:"database"`
	Show ShowInfo
	Mail MailInfo
}

// ShowInfo collects show server information
type ShowInfo struct {
	Base string
}

// DBInfo collects database connection information
type DBInfo struct {
	Server string
	Port   string
	DBname string
	User   string
	Token  string
}

// MailInfo collects email sending information
type MailInfo struct {
	Accno string
	Token string
	Peers string
}

// LoadConfigFile loads configure file
func LoadConfigFile(filename string) (*TomlConfig, error) {
	cfile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file '%s': %v", filename, err)
	}
	var conf TomlConfig
	tomlData, err := ioutil.ReadAll(cfile)
	if err != nil {
		return nil, fmt.Errorf("read file '%s': %v", filename, err)
	}
	if err := toml.Unmarshal(tomlData, &conf); err != nil {
		return nil, fmt.Errorf("decode toml data '%s': %v", string(tomlData), err)
	}
	return &conf, nil
}

// SetupDatabase connect database and ping it
func SetupDatabase(dbi *DBInfo) (*sql.DB, error) {
	cmd := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		dbi.Server, dbi.Port, dbi.DBname, dbi.User, dbi.Token)
	db, err := sql.Open("postgres", cmd)
	if err != nil {
		return nil, fmt.Errorf("open postgres[%s]: %v", cmd, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connect to postgres[%s]: %v", cmd, err)
	}
	return db, nil
}

// IsTxOpen decide whether input time is paper gold trading time
func IsTxOpen(tm *time.Time) bool {
	weekday := tm.Weekday()
	hour := tm.Hour()
	return !((weekday == time.Saturday && hour >= 4) ||
		(weekday == time.Sunday) ||
		(weekday == time.Monday && hour < 7))
}

// SendMail send email based on configure file settings
func SendMail(subject, body string, mi *MailInfo) error {
	from := mi.Accno
	to := mi.Peers
	pwd := mi.Token
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
