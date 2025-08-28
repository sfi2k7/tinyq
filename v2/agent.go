package v2

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const commandQueueListKey = "agent_command_queue"

/**
 *  command samples
 *  format:
 *  command.args...
 *
 * 	push: push.channel.item
 *  pop: pop.channel
 *  count: count.channel
 *  pause: pause.channel
 * 	resume: resume.channel
 *  status: status.channel
 *  clear: clear.channel
 *  delete: delete.channel.item
 *  empty: empty.channel
 *
 */

type agent struct {
	Id             string
	isRunning      bool
	isShuttingDown bool
}

func NewAgent() *agent {
	return &agent{
		Id:             uuid.NewString()[0:6],
		isRunning:      false,
		isShuttingDown: false,
	}
}

func extractitems(items []string) []string {
	var workitems []string
	for x := 1; x < len(items); x += 2 {
		workitems = append(workitems, items[x])
	}
	return workitems
}

func (a *agent) StartWatching(ctx context.Context, wg *sync.WaitGroup) {
	if a.isRunning {
		fmt.Println("Already running")
		return
	}

	wg.Add(1)

	fmt.Println("agent", a.Id, "started")
	a.isRunning = true

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("shutting down", a.Id)
				a.isRunning = false
				wg.Done()
				return
			default:
				items, err := Red.BLPop(context.Background(), time.Second*1, commandQueueListKey).Result()
				if err != nil {
					if !strings.Contains(err.Error(), "nil") {
						fmt.Println("some other error")
					}
					continue
				}
				workitems := extractitems(items)

				for _, item := range workitems {
					start := time.Now()
					i, err := strconv.ParseInt(item, 10, 64)
					if err != nil {
						continue
					}
					fmt.Println("result:", i*i, "in", time.Since(start).String())
				}
			}
		}
	}()
}
