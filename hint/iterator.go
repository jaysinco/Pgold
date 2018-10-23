package hint

import (
	"database/sql"
	"fmt"
	"time"
)

func newDBSource(start, end time.Time, db *sql.DB) (*dbSource, error) {
	row, err := db.Query(`
		SELECT (SELECT MIN(txtime)
				FROM pgmkt WHERE extract(epoch from txtime) >= $1),
			   (SELECT MAX(txtime) 
		        FROM pgmkt WHERE extract(epoch from txtime) <= $2)`, start.Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("query duration info from table 'pgmkt': %v", err)
	}
	defer row.Close()
	if ok := row.Next(); !ok {
		return nil, fmt.Errorf("next row empty: empty data queue")
	}
	if err := row.Scan(&start, &end); err != nil {
		return nil, fmt.Errorf("scan rows: %v", err)
	}

	row, err = db.Query(`
		SELECT txtime, bankbuy, banksell
		FROM pgmkt WHERE extract(epoch from txtime) >= $1 and extract(epoch from txtime) <= $2`, start.Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("query data from table 'pgmkt': %v", err)
	}

	return &dbSource{db, row, start, end, -1}, nil
}

type dbSource struct {
	DB     *sql.DB
	Cursor *sql.Rows
	Start  time.Time
	End    time.Time
	Length int
}

func (it *dbSource) Next() (data *price, dry bool) {
	data = new(price)
	if ok := it.Cursor.Next(); !ok {
		return nil, true
	}
	if err := it.Cursor.Scan(&data.Txtime, &data.Bankbuy, &data.Banksell); err != nil {
		return nil, true
	}
	return data, false
}

func (it *dbSource) Len() int {
	if it.Length != -1 {
		return it.Length
	}
	row, err := it.DB.Query(`
		SELECT count(*)
		FROM pgmkt WHERE extract(epoch from txtime) >= $1 and extract(epoch from txtime) <= $2`, it.Start.Unix(), it.End.Unix())
	if err != nil {
		return 0
	}
	if ok := row.Next(); !ok {
		return 0
	}
	if err := row.Scan(&it.Length); err != nil {
		return 0
	}
	return it.Length
}

func (it *dbSource) TimeRange() (start time.Time, end time.Time) {
	return it.Start, it.End
}

func (it *dbSource) Close() error {
	return it.Cursor.Close()
}

type dataIter interface {
	Len() int
	Next() (data *price, dry bool)
	TimeRange() (start time.Time, end time.Time)
	Close() error
}
