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

	strSet := pg.SplitNoSpace(pg.Config.Policy.DeploySet, ";")
	deployName := make([]string, 0)
	depolySet := make([]strategy, 0)
	for _, s := range strSet {
		for _, g := range globalPolicySet {
			if g.Name == s {
				deployName = append(deployName, s)
				depolySet = append(depolySet, g.CreateMethod(time.Now(), pg.Forever))
				break
			}
		}
	}
	if len(depolySet) == 0 {
		return fmt.Errorf("realtime: none of policy is registered: '%s'", pg.Config.Policy.DeploySet)
	}
	log.Printf("[REALTIME] deploy policy: %s", strings.Join(deployName, " | "))

	mailQueue := make(chan mail, len(depolySet))
	go func() {
		for {
			ms := <-mailQueue
			if err := pg.SendMail(ms.Subject, ms.Body, &pg.Config.Mail); err != nil {
				log.Printf("[REALTIME] [ERROR] send email '%s#%s': %v\n", ms.Subject, ms.Body, err)
			}
		}
	}()

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
		wait := make(chan policyCompleted, len(depolySet))
		for i, s := range depolySet {
			go func(id int, s strategy) {
				sig, msg := s.Dealwith(ctx)
				wait <- policyCompleted{id, sig, msg}
			}(i, s)
		}
		for n := 0; n < len(depolySet); n++ {
			pc := <-wait
			switch pc.Sig {
			case Err:
				log.Printf("[REALTIME] [ERROR] deal with `%s`: %v\n", deployName[pc.ID], pc.Msg)
			case Warn:
				sub := fmt.Sprintf("Paper gold %s.", pc.Msg)
				body := fmt.Sprintf("FROM policy `%s`", deployName[pc.ID])
				mailQueue <- mail{sub, body}
			}
		}
		time.Sleep(tick)
	}
}

type policyCompleted struct {
	ID  int
	Sig signal
	Msg string
}

type mail struct {
	Subject string
	Body    string
}
