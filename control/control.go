package control

import (
	"log"
	"strings"
	"time"

	"github.com/jaysinco/Pgold/utils"
	"github.com/urfave/cli"
)

// BatchCmd run batch subcommand
var BatchCmd = cli.Command{
	Name:  "batch",
	Usage: "Run serveral tasks simultaneously",
	Flags: []cli.Flag{
		utils.TaskListFlag,
	},
	Action: utils.InitWrapper(startRun),
}

func startRun(c *cli.Context) error {
	batch := strings.Split(c.String(utils.TaskListFlag.Name), ",")
	wait := make(chan taskCompleted)
	count := 0
	for _, task := range batch {
		task = strings.TrimSpace(task)
		for index, cmd := range c.App.Commands {
			if cmd.Name == task {
				if len(cmd.Flags) > 0 {
					log.Printf("skip has-flag task '%s'", task)
					break
				}
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
				log.Printf("skip non-exist task '%s'", task)
			}
		}
	}
	for n := 0; n < count; n++ {
		if tc := <-wait; tc.err != nil {
			log.Printf("run task '%s': %v", tc.name, tc.err)
		}
	}
	return nil
}

type taskCompleted struct {
	name string
	err  error
}
