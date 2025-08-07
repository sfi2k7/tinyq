package tinyq

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const empty = ""

type WebWorkerContext struct {
	Item    string
	Client  *WebClient
	Channel string
}

func (ctx *WebWorkerContext) NoOp() string {
	return ""
}

func (ctx *WebWorkerContext) RouteTo(channel string) string {
	return fmt.Sprintf("%s.%s", channel, ctx.Item)
}

type WebClient struct {
	url             string
	backoffduration *time.Duration
}

const Localhost = "http://localhost:8080/"

func NewWebClient(url string) *WebClient {
	//Remove the last slash if it exists
	url = strings.TrimSuffix(url, "/")

	return &WebClient{
		url: url,
	}
}

func simpleget(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *WebClient) SetBackoffDuration(duration time.Duration) {
	c.backoffduration = &duration
}

func (c *WebClient) Ack(item string) error {
	finalurl := fmt.Sprintf("%s/tinyq/ack?item=%s", c.url, item)

	_, err := simpleget(finalurl)
	if err != nil {
		return err
	}

	return nil
}

func (c *WebClient) Pop(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/pop?channel=%s", c.url, channel)
	body, err := simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body, "error") || strings.EqualFold(body, "empty") {
		return empty, nil
	}

	if err = c.Ack(string(body)); err != nil {
		return empty, err
	}

	return string(body), nil
}

func (c *WebClient) Push(item string) error {
	finalurl := fmt.Sprintf("%s/tinyq/push?item=%s", c.url, item)

	_, err := simpleget(finalurl)
	if err != nil {
		return err
	}

	return nil
}

func (c *WebClient) Pause() error {
	finalurl := fmt.Sprintf("%s/system/pause", c.url)

	if _, err := simpleget(finalurl); err != nil {
		return err
	}

	return nil
}

func (c *WebClient) Resume() error {
	finalurl := fmt.Sprintf("%s/system/resume", c.url)

	if _, err := simpleget(finalurl); err != nil {
		return err
	}

	return nil
}

func (c *WebClient) PauseStatus() (string, error) {
	finalurl := fmt.Sprintf("%s/system/status", c.url)

	body, err := simpleget(finalurl)
	if err != nil {
		return "", err
	}

	return body, nil
}

func (c *WebClient) WorkerLoop(channel string, callback func(WebWorkerContext) string) {
	fmt.Println("Starting worker on " + channel)
	ex := make(chan os.Signal, 1)
	signal.Notify(ex, os.Interrupt, syscall.SIGTERM)

	if c.backoffduration == nil {
		onesecond := time.Second
		c.backoffduration = &onesecond
	}

	for {
		select {
		case <-ex:
			fmt.Println("received signal, exiting")
			return
		default:
			item, err := c.Pop(channel)
			if err != nil {
				fmt.Println("error popping item:", err)
				continue
			}

			if item == empty || item == "error" || item == "empty" {
				time.Sleep(*c.backoffduration)
				continue
			}

			_, key := splititem(item)
			ctx := WebWorkerContext{
				Item:    key,
				Client:  c,
				Channel: channel,
			}

			func() {
				defer func() {
					if err := recover(); err != nil {
						fmt.Println("panic recovered:", err)
					}
				}()

				next := callback(ctx)
				if next != "" {
					err = c.Push(next)

					if err != nil {
						fmt.Println("error processing item:", err)
					}
				}
			}()
		}
	}
}
