package policy

import (
	"fmt"
	"log"
	"time"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// Realtime run strategy at real time
func Realtime(c *cli.Context) error {
	log.Println("[REALTIME] run")

	chosen := pg.Config.Policy.RealtimePolicy
	var stra strategy
	for _, p := range globalPolicySet {
		if p.Name == chosen {
			stra = p.CreateMethod(time.Now(), pg.Forever)
			break
		}
	}
	if stra == nil {
		return fmt.Errorf("realtime: policy name not registered: '%s'", chosen)
	}
	log.Printf("[REALTIME] policy name : %s", chosen)
	mbody := fmt.Sprintf("FROM policy `%s`", chosen)
	tick := time.Duration(pg.Config.DB.TickSec) * time.Second
	p := new(pg.Price)
	ctx := &tradeContex{p}
	tm := new(time.Time)
	for {
		if err := pg.QueryOneRow(`
			SELECT txtime, bankbuy, banksell
		    FROM pgmkt WHERE txtime = (SELECT MAX(txtime) FROM pgmkt)`,
			pg.ArgSet{}, pg.ArgSet{&tm, &p.Bankbuy, &p.Banksell}); err != nil {
			return fmt.Errorf("realtime: get current price error: %v", err)
		}
		p.Timestamp = tm.Unix()
		if sig, msg := stra.Dealwith(ctx); sig == Warn {
			sub := fmt.Sprintf("Paper gold: %s.", msg)
			if err := pg.SendMail(sub, mbody, &pg.Config.Mail); err != nil {
				return fmt.Errorf("realtime: send email: %v", err)
			}
		}
		time.Sleep(tick)
	}
}
