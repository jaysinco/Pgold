package control

import (
	"log"
	"time"

	"github.com/jaysinco/Pgold/pg"
	"github.com/urfave/cli"
)

// MutltiTask run multi tasks concurrently
func MutltiTask(c *cli.Context) error {
	log.Println("[MULTITASK] run")
	tasks := pg.SplitNoSpace(c.String(pg.FpComma(pg.TaskSetFlag.Name)), ",")
	wait := make(chan taskCompleted)
	count := 0
	for _, task := range tasks {
		for index, cmd := range c.App.Commands {
			if cmd.Name == task && len(cmd.Flags) == 0 {
				count++
				go func() {
					wait <- taskCompleted{
						cmd.Name,
						cmd.Action.(cli.ActionFunc)(c),
					}
				}()
				time.Sleep(500 * time.Millisecond)
				break
			}
			if index+1 == len(c.App.Commands) {
				log.Printf("[MULTITASK] skip non-exist/has-flag task `%s`", task)
			}
		}
	}
	for n := 0; n < count; n++ {
		if tc := <-wait; tc.err != nil {
			log.Printf("[MULTITASK] run task `%s`: %v", tc.name, tc.err)
		}
	}
	return nil
}

type taskCompleted struct {
	name string
	err  error
}
