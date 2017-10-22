package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"time"

	chart "github.com/wcharczuk/go-chart"
)

func procPaperGold(w http.ResponseWriter, r *http.Request, mlog StdLogger, db *sql.DB) {
	start, _ := time.Parse("2006-01-02 MST", time.Now().Format("2006-01-02 MST"))
	end := time.Now()
	startStr := r.Form.Get("start")
	endStr := r.Form.Get("end")
	if t, err := time.Parse("2006-01-02", startStr); startStr != "" && err == nil {
		start = t
	}
	if t, err := time.Parse("2006-01-02", endStr); endStr != "" && err == nil {
		end = t
	}
	mlog("[SEND] plot data graph within range %s ~ %s", start.Format("2006-01-02@15:04"), end.Format("2006-01-02@15:04"))
	w.Header().Set("Content-Type", "image/png")
	if err := drawPaperGoldPrice(db, start, end, w); err != nil {
		mlog("[ERROR] render graph: %v", err)
		return
	}
}

func drawPaperGoldPrice(db *sql.DB, start, end time.Time, w io.Writer) error {
	rows, err := db.Query("SELECT txtime,bankbuy FROM pgmkt WHERE txtime >= $1 AND txtime <= $2 order by txtime", start, end)
	if err != nil {
		return fmt.Errorf("query data from 'pgmkt': %v", err)
	}
	defer rows.Close()
	xval := make([]time.Time, 0)
	yval := make([]float64, 0)
	pmin := 999999.99
	pmax := -1 * pmin
	for rows.Next() {
		var txtime time.Time
		var bankbuy float64
		if err := rows.Scan(&txtime, &bankbuy); err != nil {
			return fmt.Errorf("scan rows: %v", err)
		}
		xval = append(xval, txtime)
		yval = append(yval, bankbuy)
		if bankbuy > pmax {
			pmax = bankbuy
		}
		if bankbuy < pmin {
			pmin = bankbuy
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows: %v", err)
	}

	if pmin == pmax {
		pmin -= 1
		pmax += 1
	}

	graph := chart.Chart{
		Title:      "ICBC Paper Gold Bank Buy Price",
		TitleStyle: chart.Style{Show: true},
		XAxis: chart.XAxis{
			Style:          chart.Style{Show: true},
			ValueFormatter: chart.TimeValueFormatterWithFormat("01-02#15:04"),
		},
		YAxis: chart.YAxis{
			Style: chart.Style{Show: true},
			Range: &chart.ContinuousRange{
				Max: pmax,
				Min: pmin,
			},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				Style: chart.Style{
					Show:        true,
					StrokeColor: chart.GetDefaultColor(0).WithAlpha(64),
					FillColor:   chart.GetDefaultColor(0).WithAlpha(64),
				},
				XValues: xval,
				YValues: yval,
			},
		},
	}
	return graph.Render(chart.PNG, w)
}
