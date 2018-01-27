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

func db2file(filename string) {
	host := flag.String("h", "106.15.194.234", "host for database configure")
	user := flag.String("U", "root", "user for database configure")
	dbname := flag.String("d", "root", "dbname for database configure")
	passwd := flag.String("p", "unknown", "passwords for database configure")
	flag.Parse()

	config := fmt.Sprintf("host=%s password=%s user=%s dbname=%s sslmode=disable",
		*host, *passwd, *user, *dbname)
	db, err := sql.Open("postgres", config)
	if err != nil {
		fmt.Printf("open postgres[%s]: %v\n", config, err)
		return
	}
	if err := db.Ping(); err != nil {
		fmt.Printf("connect to postgres[%s]: %v\n", config, err)
		return
	}
	defer db.Close()
	pgcs, err := queryMktData(db)
	if err != nil {
		fmt.Printf("query market data: %v\n", err)
		return
	}
	dfile, err := os.Create(filename)
	if err != nil {
		fmt.Printf("create file '%s': %v\n", filename, err)
		return
	}
	defer dfile.Close()
	fmt.Printf("writing records into file '%s'...\n", filename)
	if err := binary.Write(dfile, binary.LittleEndian, int64(len(pgcs))); err != nil {
		fmt.Printf("write record num header: %v\n", err)
		return
	}
	if err := binary.Write(dfile, binary.LittleEndian, pgcs); err != nil {
		fmt.Printf("write records: %v\n", err)
		return
	}
	fmt.Println("done!")
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
		pgc.Year = int32(txtime.Year())
		pgc.Month = int8(txtime.Month())
		pgc.Day = int8(txtime.Day())
		pgc.Hour = int8(txtime.Hour())
		pgc.Minute = int8(txtime.Minute())
		pgc.Second = int8(txtime.Second())
		pgcs = append(pgcs, pgc)
		count++
		fmt.Printf("\rrows read from database: %d", count)
	}
	fmt.Print("\n")
	return pgcs, nil
}

type pgprice struct {
	Year    int32
	Month   int8
	Day     int8
	Hour    int8
	Minute  int8
	Second  int8
	Bankbuy float32
}

func (p pgprice) String() string {
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d | %.2f",
		p.Year, p.Month, p.Day, p.Hour, p.Minute, p.Second, p.Bankbuy)
}
