package pg

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// NewSQLReader create database reader
func NewSQLReader(start, end time.Time) (Reader, error) {
	length := 0
	if err := QueryOneRow(`SELECT COUNT(*), MIN(txtime), MAX(txtime) FROM pgmkt WHERE txtime >= $1 and txtime <= $2)`,
		ArgSet{start, end}, ArgSet{&length, &start, &end}); err != nil {
		return nil, fmt.Errorf("query time range: %v", err)
	}
	rows, err := DB.Query(`SELECT txtime, bankbuy, banksell FROM pgmkt WHERE txtime >= $1 and txtime <= $2`, start, end)
	if err != nil {
		return nil, fmt.Errorf("select from table 'pgmkt': %v", err)
	}
	return &sqlReader{DB, rows, start, end, length}, nil
}

type sqlReader struct {
	DB     *sql.DB
	Rows   *sql.Rows
	Start  time.Time
	End    time.Time
	Length int
}

func (r *sqlReader) Read(p *Price) (dry bool, err error) {
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

func (r *sqlReader) ReadAll(set []*Price) error {
	count := 0
	for {
		if count >= len(set) {
			return nil
		}
		p := new(Price)
		if dry, err := r.Read(p); err != nil {
			return fmt.Errorf("read next one: %v", err)
		} else if dry {
			return nil
		} else {
			set[count] = p
			count++
		}
	}
}

func (r *sqlReader) Len() int {
	return r.Length
}

func (r *sqlReader) TimeRange() (start time.Time, end time.Time) {
	return r.Start, r.End
}

func (r *sqlReader) Close() error {
	return r.Rows.Close()
}

// NewSQLWriter create database writer
func NewSQLWriter() Writer {
	return &sqlWriter{DB}
}

type sqlWriter struct {
	DB *sql.DB
}

func (w *sqlWriter) Write(p *Price) error {
	_, err := w.DB.Exec("insert into pgmkt(txtime,bankbuy,banksell) values($1,$2,$3)",
		time.Unix(p.Timestamp, 0), p.Bankbuy, p.Banksell)
	return err
}

func (w *sqlWriter) WriteAll(set []*Price) (n int, err error) {
	tx, err := w.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("tx begin: %v", err)
	}
	defer tx.Commit()
	stmt, err := tx.Prepare("insert into pgmkt(txtime,bankbuy,banksell) values($1,$2,$3)")
	if err != nil {
		return 0, fmt.Errorf("tx prepare: %v", err)
	}
	defer stmt.Close()
	for _, p := range set {
		if _, err = stmt.Exec(time.Unix(p.Timestamp, 0), p.Bankbuy, p.Banksell); err == nil {
			n++
		}
	}
	return n, nil
}

func (w *sqlWriter) Close() error {
	return nil
}

// NewBinaryReader create binary reader
func NewBinaryReader(source io.Reader) (Reader, error) {
	length := 0
	if err := binary.Read(source, binary.LittleEndian, &length); err != nil {
		return nil, fmt.Errorf("read length header: %v", err)
	}
	buf := make([]*Price, length)
	if err := binary.Read(source, binary.LittleEndian, buf); err != nil {
		return nil, fmt.Errorf("read price: %v", err)
	}
	var start, end int64
	if length > 0 {
		start, end = buf[0].Timestamp, buf[len(buf)-1].Timestamp
		for _, p := range buf {
			if p.Timestamp > end {
				end = p.Timestamp
			}
			if p.Timestamp < start {
				start = p.Timestamp
			}
		}
	}
	return &binaryReader{source, buf, time.Unix(start, 0), time.Unix(end, 0), length, -1}, nil
}

type binaryReader struct {
	Source io.Reader
	Buffer []*Price
	Start  time.Time
	End    time.Time
	Length int
	Pos    int
}

func (r *binaryReader) Read(p *Price) (dry bool, err error) {
	r.Pos++
	if r.Pos >= r.Length {
		return true, nil
	}
	*p = *r.Buffer[r.Pos]
	return false, nil
}

func (r *binaryReader) ReadAll(set []*Price) error {
	for i := 0; i < len(set); i++ {
		r.Pos++
		if r.Pos >= r.Length {
			return nil
		}
		set[i] = r.Buffer[r.Pos]
	}
	return nil
}

func (r *binaryReader) Len() int {
	return r.Length
}

func (r *binaryReader) TimeRange() (start, end time.Time) {
	return
}

func (r *binaryReader) Close() error {
	return nil
}

// NewBinaryWriter create binary writer
func NewBinaryWriter(dest io.Writer) (Writer, error) {
	return &binaryWriter{dest, make([]*Price, 0)}, nil
}

type binaryWriter struct {
	Dest   io.Writer
	Buffer []*Price
}

func (w *binaryWriter) Write(p *Price) error {
	w.Buffer = append(w.Buffer, p)
	return nil
}

func (w *binaryWriter) WriteAll(set []*Price) (n int, err error) {
	w.Buffer = append(w.Buffer, set...)
	return len(set), nil
}

func (w *binaryWriter) Close() error {
	length := len(w.Buffer)
	if err := binary.Write(w.Dest, binary.LittleEndian, length); err != nil {
		return fmt.Errorf("write length header: %v", err)
	}
	if err := binary.Write(w.Dest, binary.LittleEndian, w.Buffer); err != nil {
		return fmt.Errorf("write price: %v", err)
	}
	return nil
}

// CreateMktTbl create market table
func CreateMktTbl() error {
	_, err := DB.Exec(`create table if not exists pgmkt (
		txtime    timestamp(0) with time zone primary key,
		bankbuy   numeric(8,2),
		banksell  numeric(8,2)
	)`)
	return fmt.Errorf("create table 'pgmkt': %v", err)
}

// Price contains paper gold price tick info
type Price struct {
	Timestamp int64
	Bankbuy   float32
	Banksell  float32
}

func (p Price) String() string {
	tm := time.Unix(p.Timestamp, 0)
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d | %.2f | %.2f",
		tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), p.Bankbuy, p.Banksell)
}

// Reader read price from source
type Reader interface {
	Read(p *Price) (dry bool, err error)
	ReadAll(set []*Price) error
	Len() int
	TimeRange() (start, end time.Time)
	Close() error
}

// Writer writer price into source
type Writer interface {
	Write(p *Price) error
	WriteAll(set []*Price) (n int, err error)
	Close() error
}
