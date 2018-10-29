package pg

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

// global variables
var (
	DB        *sql.DB
	DBSTR     string
	Config    *TomlConfig
	SourceDir = filepath.ToSlash(os.Getenv("GOPATH")) + "/src/github.com/jaysinco/Pgold"
	StampFmt  = "06/01/02 15:04:05"
	Forever   = time.Now().Add(24 * time.Hour * 365 * 100)
)

// flags
var (
	ConfigFlag = cli.StringFlag{
		Name:  "config,c",
		Value: SourceDir + "/pgold.conf",
		Usage: "load configuration from `FILE`",
	}
	InfileFlag = cli.StringFlag{
		Name:  "in,i",
		Value: "pg" + time.Now().Format("060102") + ".dat",
		Usage: "read input from `FILE`",
	}
	OutfileFlag = cli.StringFlag{
		Name:  "out,o",
		Value: "pg" + time.Now().Format("060102") + ".dat",
		Usage: "write output into `FILE`",
	}
	OnlyTxOpenFlag = cli.BoolFlag{
		Name:  "tx-only,x",
		Usage: "only when transaction open",
	}
	TaskSetFlag = cli.StringFlag{
		Name:  "task,t",
		Value: "market, server, realtime",
		Usage: "run multi tasks concurrently as per `LIST`",
	}
	StartDateFlag = cli.StringFlag{
		Name:  "start,s",
		Value: "171019",
		Usage: "start from `DATE`",
	}
	EndDateFlag = cli.StringFlag{
		Name:  "end,e",
		Value: time.Now().Add(24 * time.Hour).Format("060102"),
		Usage: "end by `DATE`",
	}
	PolicyFlag = cli.StringFlag{
		Name:  "policy,p",
		Value: "RandomTrader",
		Usage: "strategy `Name`",
	}
)

// SetupConfig loads configure file
func SetupConfig(filename string) (*TomlConfig, error) {
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
	cmd := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
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

// FpComma get first part in a comma seperated string
func FpComma(s string) string {
	return strings.Split(s, ",")[0]
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

// Setup loads configure file and setup database
func Setup(cmdAction cli.ActionFunc) cli.ActionFunc {
	return func(c *cli.Context) (err error) {
		Config, err = SetupConfig(c.GlobalString(FpComma(ConfigFlag.Name)))
		if err != nil {
			return fmt.Errorf("load configure file: %v", err)
		}

		DB, err = SetupDatabase(&Config.DB)
		DBSTR = fmt.Sprintf("postgres@%s:%d/%s",
			Config.DB.Server, Config.DB.Port, Config.DB.DBname)
		if err != nil {
			return fmt.Errorf("setup database: %v", err)
		}
		return cmdAction(c)
	}
}

// ArgSet represents list of arguments
type ArgSet []interface{}

// QueryOneRow is a handy way to query just one row from database
func QueryOneRow(sql string, query, dest ArgSet) error {
	row, err := DB.Query(sql, query...)
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

// IsTradeOpen decide whether input time is paper gold trading time
func IsTradeOpen(tm time.Time) bool {
	weekday := tm.Weekday()
	hour := tm.Hour()
	return !((weekday == time.Saturday && hour >= 4) ||
		(weekday == time.Sunday) ||
		(weekday == time.Monday && hour < 7))
}

// ParseDate parse YYMMDD based on CST time zone
func ParseDate(yymmdd string) (time.Time, error) {
	return time.Parse("060102 MST", yymmdd+" CST")
}

// TomlConfig stands for configure file
type TomlConfig struct {
	DB     DBInfo `toml:"database"`
	Server ServerInfo
	Mail   MailInfo
	Policy PolicyInfo
}

// ServerInfo collects show server information
type ServerInfo struct {
	Port    int
	Basedir string
}

// DBInfo collects database connection information
type DBInfo struct {
	TickSec int
	Server  string
	Port    int
	DBname  string
	User    string
	Token   string
}

// MailInfo collects email sending information
type MailInfo struct {
	Accno string
	Token string
	Peers string
}

// PolicyInfo collects policy parameters information
type PolicyInfo struct {
	DeploySet           string
	SysBrokenMin        int
	RandSeed            int64
	RandTradeFreqPerDay float64
	WaveThreshold       float32
	WaveIntervalMin     int
}
