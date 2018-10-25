package pg

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

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

// Reader read price from source
type Reader interface {
	ReadOne(p *Price) (dry bool, err error)
	Read(set []*Price) error
	Len() int
	TimeRange() (start, end time.Time)
	Close() error
}

// Writer writer price into source
type Writer interface {
	WriteOne(p *Price) error
	Write(set []*Price) (n int, err error)
	Close() error
}

// NewDBReader create database reader
func NewDBReader(start, end time.Time) (Reader, error) {
	length := 0
	if err := QueryOneRow(`SELECT COUNT(*), MIN(txtime), MAX(txtime) FROM pgmkt WHERE txtime >= $1 and txtime <= $2)`,
		ArgSet{start, end}, ArgSet{&length, &start, &end}); err != nil {
		return nil, fmt.Errorf("query time range: %v", err)
	}
	rows, err := DB.Query(`SELECT txtime, bankbuy, banksell FROM pgmkt WHERE txtime >= $1 and txtime <= $2`, start, end)
	if err != nil {
		return nil, fmt.Errorf("select from table 'pgmkt': %v", err)
	}
	return &dbReader{DB, rows, start, end, length}, nil
}

// NewDBWriter create database writer
func NewDBWriter() Writer {
	return &dbWriter{DB}
}

// NewBinFileReader create binary file reader
func NewBinFileReader(filename string) (Reader, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %v", err)
	}
	length := new(int)
	if err := binary.Read(fh, binary.LittleEndian, length); err != nil {
		return nil, fmt.Errorf("read length header: %v", err)
	}
	return &binFileReader{fh, filename, *length}, nil
}

// NewBinFileWriter create binary file writer
func NewBinFileWriter(filename string) (Writer, error) {
	fh, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("create file: %v", err)
	}
	return &binFileWriter{fh, filename}, nil
}

// IsTradeOpen decide whether input time is paper gold trading time
func IsTradeOpen(tm time.Time) bool {
	weekday := tm.Weekday()
	hour := tm.Hour()
	return !((weekday == time.Saturday && hour >= 4) ||
		(weekday == time.Sunday) ||
		(weekday == time.Monday && hour < 7))
}

type dbReader struct {
	DB     *sql.DB
	Rows   *sql.Rows
	Start  time.Time
	End    time.Time
	Length int
}

func (r *dbReader) ReadOne(p *Price) (dry bool, err error) {
	if ok := r.Rows.Next(); !ok {
		return true, nil
	}
	tm := new(time.Time)
	if err := r.Rows.Scan(tm, &p.Bankbuy, &p.Banksell); err != nil {
		return true, fmt.Errorf("scan row: %v", err)
	}
	p.Timestamp = tm.Unix()
	return false, nil
}

func (r *dbReader) Read(set []*Price) error {
	count := 0
	for {
		if count >= len(set) {
			return nil
		}
		p := new(Price)
		if dry, err := r.ReadOne(p); err != nil {
			return fmt.Errorf("read next one: %v", err)
		} else if dry {
			return nil
		} else {
			set[count] = p
			count++
		}
	}
}

func (r *dbReader) Len() int {
	return r.Length
}

func (r *dbReader) TimeRange() (start time.Time, end time.Time) {
	return r.Start, r.End
}

func (r *dbReader) Close() error {
	return r.Rows.Close()
}

type dbWriter struct {
	DB *sql.DB
}

func (w *dbWriter) WriteOne(p *Price) error {
	_, err := w.DB.Exec("insert into pgmkt(txtime,bankbuy,banksell) values($1,$2,$3)",
		time.Unix(p.Timestamp, 0), p.Bankbuy, p.Banksell)
	return err
}

func (w *dbWriter) Write(set []*Price) (n int, lstErr error) {
	for _, p := range set {
		if lstErr = w.WriteOne(p); lstErr == nil {
			n++
		}
	}
	return
}

func (w *dbWriter) Close() error {
	return nil
}

type binFileReader struct {
	FileHandler *os.File
	FileName    string
	Length      int
}

func (r *binFileReader) ReadOne(p *Price) (dry bool, err error) {
	if err = binary.Read(r.FileHandler, binary.LittleEndian, p); err != nil {
		return true, nil
	}
	return false, nil
}

func (r *binFileReader) Read(set []*Price) error {
	return binary.Read(r.FileHandler, binary.LittleEndian, set)
}

func (r *binFileReader) Len() int {
	return r.Length
}
func (r *binFileReader) TimeRange() (start, end time.Time) {
	return
}
func (r *binFileReader) Close() error {
	return r.FileHandler.Close()
}

type binFileWriter struct {
	FileHandler *os.File
	FileName    string
}

func (w *binFileWriter) WriteOne(p *Price) error {

}
func (w *binFileWriter) Write(set []*Price) (n int, err error) {
	length := len(set)
	if err := binary.Write(w.FileHandler, binary.LittleEndian, length); err != nil {
		return 0, fmt.Errorf("write length header: %v", err)
	}
	if err := binary.Write(w.FileHandler, binary.LittleEndian, set); err != nil {
		return 0, fmt.Errorf("write price: %v", err)
	}
	return length, nil
}

func (w *binFileWriter) Close() error {
	return w.FileHandler.Close()
}
