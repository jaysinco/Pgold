package hint

import (
	"database/sql"
	"math/rand"
	"time"
)

func newRandTester(seeds int64) *randomTester {
	rnd := rand.New(rand.NewSource(seeds))
	return &randomTester{rnd, Pass, 0}
}

type randomTester struct {
	Rnd     *rand.Rand
	LastSig signal
	Count   int
}

func (s *randomTester) Name() string {
	return "random tester"
}

func (s *randomTester) Dealwith(ctx *tradeContex) (sig signal, msg string) {
	if s.Rnd.Float32() < 0.001 {
		s.Count++
		if s.Count <= 8 {
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
	Name() string
	Dealwith(ctx *tradeContex) (sig signal, msg string)
}

type tradeContex struct {
	*price
	DB *sql.DB
}

type price struct {
	Txtime   time.Time
	Bankbuy  float32
	Banksell float32
}
type signal int

// action type
const (
	Pass signal = iota
	Buy
	Sell
	Warn
)
