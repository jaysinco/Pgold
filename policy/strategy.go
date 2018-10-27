package policy

import (
	"math"
	"math/rand"
	"time"

	"github.com/jaysinco/Pgold/pg"
)

var globalPolicySet = []policy{
	{"RandomTrader", newRandomTrader},
}

func newRandomTrader(start, end time.Time) strategy {
	trader := new(randomTrader)
	trader.Rnd = rand.New(rand.NewSource(pg.Config.Policy.Seed))
	trader.LastSig = Pass
	sub := end.Sub(start)
	trader.MaxTran = int(math.Floor(pg.Config.Policy.TradeFreqPerDay * sub.Hours() / 24))
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

func (s *randomTrader) Dealwith(ctx *tradeContex) (sig signal, msg string) {
	if s.Rnd.Float64() < s.Prob {
		s.Count++
		if s.Count <= s.MaxTran {
			switch s.LastSig {
			case Pass:
				sig = Buy
			case Buy:
				sig = Sell
			case Sell:
				sig = Buy
			}
			s.LastSig = sig
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
