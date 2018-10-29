package policy

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// Realtime run strategy at real time
func Realtime(c *cli.Context) error {
	log.Println("[REALTIME] run")

	deploySet := strings.Split(pg.Config.Policy.DeploySet, ";")
	deployDesc := make([]string, 0)
	var plc []strategy
	for _, pld := range deploySet {
		pln := strings.TrimSpace(pld)
		for _, plb := range globalPolicySet {
			if plb.Name == pln {
				deployDesc = append(deployDesc, pln)
				plc = append(plc, plb.CreateMethod(time.Now(), pg.Forever))
				break
			}
		}
	}
	if len(plc) == 0 {
		return fmt.Errorf("realtime: none of policy is registered: '%s'", pg.Config.Policy.DeploySet)
	}
	log.Printf("[REALTIME] deploy policy: %s", strings.Join(deployDesc, " | "))

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
		for i, s := range plc {
			if sig, msg := s.Dealwith(ctx); sig == Warn {
				sub := fmt.Sprintf("Paper gold %s.", msg)
				mbody := fmt.Sprintf("FROM policy `%s`", deployDesc[i])
				if err := pg.SendMail(sub, mbody, &pg.Config.Mail); err != nil {
					return fmt.Errorf("realtime: send email: %v", err)
				}
			}
		}
		time.Sleep(tick)
	}
}
