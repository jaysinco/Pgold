package policy

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jaysinco/Pgold/pg"
)

var globalPolicySet = []policy{
	{"SystemKeeper", newSystemKeeper},
	{"RandomTrader", newRandomTrader},
	{"WaveCaptor", newWaveCaptor},
}

func newSystemKeeper(start, end time.Time) strategy {
	keeper := new(systemKeeper)
	keeper.BrokenMin = time.Duration(pg.Config.Policy.SysBrokenMin) * time.Minute
	return keeper
}

type systemKeeper struct {
	BrokenMin  time.Duration
	LastWarnTm time.Time
}

func (k *systemKeeper) Dealwith(ctx *tradeContex) (sig signal, msg string) {
	systm := time.Now()
	latest := time.Unix(ctx.Timestamp, 0)
	if systm.Sub(k.LastWarnTm) > k.BrokenMin && systm.Sub(latest) >= k.BrokenMin {
		k.LastWarnTm = systm
		return Warn, fmt.Sprintf("market data not updated since %s", latest.Format(pg.StampFmt))
	}
	return Pass, ""
}

func newWaveCaptor(start, end time.Time) strategy {
	captor := new(waveCaptor)
	captor.Threshold = pg.Config.Policy.WaveThreshold
	captor.Interval = time.Duration(pg.Config.Policy.WaveIntervalMin) * time.Minute
	return captor
}

type waveCaptor struct {
	Threshold  float32
	Interval   time.Duration
	LastWarnTm time.Time
}

func (w *waveCaptor) Dealwith(ctx *tradeContex) (sig signal, msg string) {
	now := time.Unix(ctx.Timestamp, 0)
	if now.Sub(w.LastWarnTm) <= w.Interval {
		return Pass, ""
	}
	bef := now.Add(-1 * w.Interval)
	var maxbuy, minbuy float32
	var maxbuytm, minbuytm time.Time
	if err := pg.QueryOneRow(`
		SELECT (SELECT MIN(b.txtime) FROM pgmkt b WHERE b.txtime >= $1 and b.txtime <= $2 and b.bankbuy = MAX(a.bankbuy)) maxbuytm,
			   MAX(a.bankbuy) maxbuy, 
			   (SELECT MIN(b.txtime) FROM pgmkt b WHERE b.txtime >= $1 and b.txtime <= $2 and b.bankbuy = MIN(a.bankbuy)) minbuytm,
			   MIN(a.bankbuy) minbuy
		FROM pgmkt a WHERE a.txtime >= $1 and a.txtime <= $2`,
		pg.ArgSet{bef, now}, pg.ArgSet{&maxbuytm, &maxbuy, &minbuytm, &minbuy}); err != nil {
		return Warn, fmt.Sprintf("[ERROR] get max/min buy error: %v", err)
	}
	if ctx.Bankbuy-minbuy > w.Threshold {
		w.LastWarnTm = now
		return Warn, fmt.Sprintf("rise %.2f RMB/g during last %d minutes",
			ctx.Bankbuy-minbuy, int(now.Sub(minbuytm)/time.Minute))
	}
	if maxbuy-ctx.Bankbuy > w.Threshold {
		w.LastWarnTm = now
		return Warn, fmt.Sprintf("down %.2f RMB/g during last %d minutes",
			maxbuy-ctx.Bankbuy, int(now.Sub(maxbuytm)/time.Minute))
	}
	return Pass, ""
}

func newRandomTrader(start, end time.Time) strategy {
	trader := new(randomTrader)
	trader.Rnd = rand.New(rand.NewSource(pg.Config.Policy.RandSeed))
	trader.LastSig = Pass
	sub := end.Sub(start)
	trader.MaxTran = int(math.Floor(pg.Config.Policy.RandTradeFreqPerDay * sub.Hours() / 24))
	trader.Prob = float64(trader.MaxTran) / (sub.Minutes() * 2)
	return trader
}

type randomTrader struct {
	Rnd     *rand.Rand
	LastSig signal
	MaxTran int
	Prob    float64
	Count   int
}

func (r *randomTrader) Dealwith(ctx *tradeContex) (sig signal, msg string) {
	if r.Rnd.Float64() < r.Prob {
		r.Count++
		if r.Count <= r.MaxTran {
			switch r.LastSig {
			case Pass:
				sig = Buy
			case Buy:
				sig = Sell
			case Sell:
				sig = Buy
			}
			r.LastSig = sig
		}
	}
	return
}

type strategy interface {
	Dealwith(ctx *tradeContex) (sig signal, msg string)
}

type policy struct {
	Name         string
	CreateMethod func(start, end time.Time) strategy
}

type tradeContex struct {
	*pg.Price
}

type signal int

// action type
const (
	Pass signal = iota
	Buy
	Sell
	Warn
)
