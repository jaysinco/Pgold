package main

import (
	"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	filename := "pgmkt.dat"
	// fmt.Println(exportMktData(filename))
	fmt.Println(checkDataExport(filename))
}

func checkDataExport(filename string) error {
	pgcs, err := readMktData(filename)
	if err != nil {
		return fmt.Errorf("read market data from '%s': %v", filename, err)
	}
	fmt.Printf("%d records read from '%s'\n", len(pgcs), filename)
	for i := 0; i < 5; i++ {
		fmt.Println(pgcs[i])
	}
	return nil
}

func readMktData(filename string) ([]pgprice, error) {
	dfile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file '%s': %v", filename, err)
	}
	var num int64
	err = binary.Read(dfile, binary.LittleEndian, &num)
	if err != nil {
		return nil, fmt.Errorf("read record num header: %v", err)
	}
	pgcs := make([]pgprice, num)
	err = binary.Read(dfile, binary.LittleEndian, pgcs)
	if err != nil {
		return nil, fmt.Errorf("read records: %v", err)
	}
	return pgcs, nil
}

func exportMktData(filename string) error {
	host := flag.String("h", "127.0.0.1", "host for database configure")
	user := flag.String("U", "root", "user for database configure")
	dbname := flag.String("d", "root", "dbname for database configure")
	passwd := flag.String("p", "unknown", "passwords for database configure")
	flag.Parse()

	config := fmt.Sprintf("host=%s password=%s user=%s dbname=%s sslmode=disable",
		*host, *passwd, *user, *dbname)
	db, err := sql.Open("postgres", config)
	if err != nil {
		return fmt.Errorf("open postgres[%s]: %v", config, err)
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("connect to postgres[%s]: %v", config, err)
	}
	defer db.Close()

	pgcs, err := queryMktData(db)
	if err != nil {
		return fmt.Errorf("query market data: %v", err)
	}
	dfile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file '%s': %v", filename, err)
	}
	defer dfile.Close()
	num := len(pgcs)
	if err := binary.Write(dfile, binary.LittleEndian, int64(num)); err != nil {
		return fmt.Errorf("write record num header: %v", err)
	}
	if err := binary.Write(dfile, binary.LittleEndian, pgcs); err != nil {
		return fmt.Errorf("write records: %v", err)
	}
	fmt.Printf("%d records written into '%s'\n", num, filename)
	return nil
}

func queryMktData(db *sql.DB) ([]pgprice, error) {
	rows, err := db.Query("SELECT txtime,bankbuy FROM pgmkt order by txtime")
	if err != nil {
		return nil, fmt.Errorf("query data from table 'pgmkt': %v", err)
	}
	defer rows.Close()
	txtime := new(time.Time)
	pgcs := make([]pgprice, 0)
	count := 0
	for rows.Next() {
		var pgc pgprice
		if err := rows.Scan(&txtime, &pgc.Bankbuy); err != nil {
			return nil, fmt.Errorf("scan rows: %v", err)
		}
		if isTxOpen(txtime) {
			pgc.Timestamp = txtime.Unix()
			pgcs = append(pgcs, pgc)
		}
		count++
		fmt.Printf("\r%d rows read from database", count)
	}
	fmt.Print("\n")
	return pgcs, nil
}

func isTxOpen(tm *time.Time) bool {
	weekday := tm.Weekday()
	hour := tm.Hour()
	return !((weekday == time.Saturday && hour >= 4) ||
		(weekday == time.Sunday) ||
		(weekday == time.Monday && hour < 7))
}

type pgprice struct {
	Timestamp int64
	Bankbuy   float32
}

func (p pgprice) String() string {
	tm := time.Unix(p.Timestamp, 0)
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d | %.2f",
		tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), p.Bankbuy)
}
