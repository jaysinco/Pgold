package utils

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

// DataIter protocol
type DataIter interface {
	Len() int
	Range() (start time.Time, end time.Time)
	Next() (data *Price, dry bool)
	Close() error
}

// NewIterFromDatabase create database iterator
func NewIterFromDatabase(start, end time.Time) (DataIter, error) {
	length := 0
	if err := QueryOne(`SELECT COUNT(*), MIN(txtime), MAX(txtime) FROM pgmkt WHERE txtime >= $1 and txtime <= $2)`,
		Qargs{start, end}, Qargs{&length, &start, &end}); err != nil {
		return nil, fmt.Errorf("query time range: %v", err)
	}
	rows, err := DB.Query(`SELECT txtime, bankbuy, banksell FROM pgmkt WHERE txtime >= $1 and txtime <= $2`, start, end)
	if err != nil {
		return nil, fmt.Errorf("select from table 'pgmkt': %v", err)
	}
	return &dbSource{DB, rows, start, end, length}, nil
}

type dbSource struct {
	DB     *sql.DB
	Rows   *sql.Rows
	Start  time.Time
	End    time.Time
	Length int
}

func (it *dbSource) Next() (data *Price, dry bool) {
	data = new(Price)
	if ok := it.Rows.Next(); !ok {
		return nil, true
	}
	var tm time.Time
	if err := it.Rows.Scan(&tm, &data.Bankbuy, &data.Banksell); err != nil {
		return nil, true
	}
	data.Timestamp = tm.Unix()
	return data, false
}

func (it *dbSource) Len() int {
	return it.Length
}

func (it *dbSource) Range() (start time.Time, end time.Time) {
	return it.Start, it.End
}

func (it *dbSource) Close() error {
	return it.Rows.Close()
}

// IsTxOpen decide whether input time is paper gold trading time
func IsTxOpen(tm time.Time) bool {
	weekday := tm.Weekday()
	hour := tm.Hour()
	return !((weekday == time.Saturday && hour >= 4) ||
		(weekday == time.Sunday) ||
		(weekday == time.Monday && hour < 7))
}

// PriceList is set of price
type PriceList []*Price

// GetPriceFromDB collect price from database into list
func GetPriceFromDB(start, end time.Time, onlyTxOpen, print bool) (PriceList, error) {
	it, err := NewIterFromDatabase(start, end)
	if err != nil {
		return nil, fmt.Errorf("create new databse iterator: %v", err)
	}
	defer it.Close()
	total := it.Len()
	pgcs := make(PriceList, 0)
	count := 0
	for {
		data, dry := it.Next()
		if dry {
			break
		}
		if (!onlyTxOpen) || IsTxOpen(time.Unix(data.Timestamp, 0)) {
			pgcs = append(pgcs, data)
		}
		count++
		if print && count%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(count)/float32(total))
		}
	}
	if print {
		fmt.Print("\r")
	}
	return pgcs, nil
}

//InsertPriceIntoDB insert price into database
func InsertPriceIntoDB(pgcs PriceList, print bool) (success int, err error) {
	stmt, err := DB.Prepare("insert into pgmkt(txtime,bankbuy,banksell) values($1,$2,$3)")
	if err != nil {
		return 0, fmt.Errorf("prepare sql: %v", err)
	}
	for index, pgc := range pgcs {
		if print && index%100 == 0 {
			fmt.Printf("\r >> %.1f%%", float32(index)/float32(len(pgcs))*100)
		}
		_, err = stmt.Exec(time.Unix(pgc.Timestamp, 0), pgc.Bankbuy, pgc.Banksell)
		if err != nil {
			continue
		} else {
			success++
		}
	}
	if print {
		fmt.Print("\r")
	}
	return success, nil
}

// GetPriceFromBinFile collect price from binary file into list
func GetPriceFromBinFile(filename string) (PriceList, error) {
	dfile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file '%s': %v", filename, err)
	}
	defer dfile.Close()
	var num int64
	err = binary.Read(dfile, binary.LittleEndian, &num)
	if err != nil {
		return nil, fmt.Errorf("read record num header: %v", err)
	}
	pgcs := make(PriceList, num)
	err = binary.Read(dfile, binary.LittleEndian, pgcs)
	if err != nil {
		return nil, fmt.Errorf("read records: %v", err)
	}
	return pgcs, nil
}

// WritePriceIntoBinFile write price from list into binary file
func WritePriceIntoBinFile(filename string, pgcs PriceList) error {
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
	return nil
}
