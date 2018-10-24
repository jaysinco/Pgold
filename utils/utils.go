package utils

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
)

// shared variables
var (
	DB        *sql.DB
	Config    *TomlConfig
	SourceDir = filepath.ToSlash(os.Getenv("GOPATH")) + "/src/github.com/jaysinco/Pgold"
)

// general settings
var (
	ConfigFlag = cli.StringFlag{
		Name:  "config,c",
		Value: SourceDir + "/pgold.conf",
		Usage: "load configuration from `FILE`",
	}
	InfileFlag = cli.StringFlag{
		Name:  "in,i",
		Value: "pgmkt_" + time.Now().Format("060102") + ".dat",
		Usage: "read input from `FILE`",
	}
	OutfileFlag = cli.StringFlag{
		Name:  "out,o",
		Value: "pgmkt_" + time.Now().Format("060102") + ".dat",
		Usage: "write output into `FILE`",
	}
	OnlyTxOpenFlag = cli.BoolFlag{
		Name:  "tx-only,x",
		Usage: "only when transaction open",
	}
	TaskListFlag = cli.StringFlag{
		Name:  "task,t",
		Value: "market, show, hint",
		Usage: "run batch as per `LIST`",
	}
	StartDateFlag = cli.StringFlag{
		Name:  "start,s",
		Value: "171019",
		Usage: "start from date",
	}
	EndDateFlag = cli.StringFlag{
		Name:  "end,e",
		Value: time.Now().Add(24 * time.Hour).Format("060102"),
		Usage: "end by date",
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
	Port    string
	Basedir string
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

// Price contains paper gold price tick info
type Price struct {
	Timestamp int64   `json:"t"`
	Bankbuy   float32 `json:"p"`
	Banksell  float32 `json:"s,omitempty"`
}

func (p Price) String() string {
	tm := time.Unix(p.Timestamp, 0)
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d | %.2f | %.2f",
		tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), p.Bankbuy, p.Banksell)
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

// GetFlagName used for flag alias name lookup
func GetFlagName(flag cli.Flag) string {
	return strings.Split(flag.GetName(), ",")[0]
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

// InitWrapper do initialize stuff
func InitWrapper(cmdAction cli.ActionFunc) cli.ActionFunc {
	return func(c *cli.Context) (err error) {
		Config, err = LoadConfigFile(c.GlobalString(GetFlagName(ConfigFlag)))
		if err != nil {
			return fmt.Errorf("load configure file: %v", err)
		}

		DB, err = SetupDatabase(&Config.DB)
		if err != nil {
			return fmt.Errorf("setup database: %v", err)
		}
		return cmdAction(c)
	}
}

// Qargs represents list of args
type Qargs []interface{}

// QueryOne is a handy way to query just one row from database
func QueryOne(query string, args, dest Qargs) error {
	row, err := DB.Query(query, args...)
	if err != nil {
		return fmt.Errorf("query: %v", err)
	}
	defer row.Close()
	if ok := row.Next(); !ok {
		return fmt.Errorf("next row: empty data queue")
	}
	if err := row.Scan(dest...); err != nil {
		return fmt.Errorf("scan row: %v", err)
	}
	return nil
}

// ParseDate parse YYMMDD based on CST time zone
func ParseDate(yymmdd string) (time.Time, error) {
	return time.Parse("060102 MST", yymmdd+" CST")
}
