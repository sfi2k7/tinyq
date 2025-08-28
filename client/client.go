package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sfi2k7/tinyq"
)

type TqWorker func(*WebWorkerContext) string

const empty = ""

var notiteminqueue = errors.New("no item in queue")

func serialize(data map[string]string) string {
	databytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(databytes)
	return encoded
}

func serializepairs(data ...any) string {
	ctx := WebWorkerContext{}
	ctx.addpairstodata(data...)
	if ctx.Data == nil && len(ctx.Data) == 0 {
		return ""
	}
	return serialize(ctx.Data)
}

func deserialize(data string) map[string]string {
	var m = make(map[string]string, 0)
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return m
	}

	err = json.Unmarshal(decoded, &m)
	if err != nil {
		return nil
	}
	return m
}

type WebClient struct {
	url             string
	backoffduration *time.Duration
	token           string
	appname         string
}

const Localhost = "http://localhost:8080/"

type Option func(*WebClient)

func WithUrl(url string) Option {
	url = strings.TrimSuffix(url, "/")
	return func(c *WebClient) {
		c.url = url
	}
}

func WithToken(token string) Option {
	return func(c *WebClient) {
		c.token = token
	}
}

func WithBackoffDuration(duration time.Duration) Option {
	return func(c *WebClient) {
		c.backoffduration = &duration
	}
}

func WithAppname(appname string) Option {
	return func(c *WebClient) {
		c.appname = appname
	}
}

func WithOptions(options ...Option) Option {
	return func(c *WebClient) {
		for _, option := range options {
			option(c)
		}
	}
}

func NewWebClient(options ...Option) *WebClient {

	twoseconds := time.Second * 2
	c := &WebClient{
		url:             "http://localhost:8080",
		appname:         "default",
		backoffduration: &twoseconds,
	}

	for _, option := range options {
		option(c)
	}

	return c
}

func (c *WebClient) SetAppname(name string) {
	c.appname = name
}

func (c *WebClient) SetUrl(url string) {
	url = strings.TrimSuffix(url, "/")
	c.url = url
}

func (c *WebClient) SetToken(token string) {
	c.token = token
}

func (c *WebClient) SetBackoffDuration(duration time.Duration) {
	c.backoffduration = &duration
}

type response struct {
	Message string `json:"message"`
	Error   string `json:"error"`
	Took    string `json:"took"`
}

func (c *WebClient) simpleget(remote string) (*response, error) {
	parsed, _ := url.Parse(remote)

	query := parsed.Query()
	if len(c.token) > 0 {
		query.Set("token", c.token)
	}

	if len(c.appname) > 0 {
		query.Set("app", c.appname)
	}

	parsed.RawQuery = query.Encode()
	remote = parsed.String()

	// fmt.Println("GETing on:", url)
	start := time.Now()
	req, err := http.NewRequest("GET", remote, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	// fmt.Println("making request", url)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("readall", err)
		return nil, err
	}

	if bytes.HasPrefix(body, []byte("[")) {
		return &response{Took: time.Since(start).String(), Message: string(body)}, nil
	}

	// fmt.Println("res", string(body))
	var res response
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("body", string(body), "url", remote)
		fmt.Println("unmarshal", err)
		return nil, err
	}

	// fmt.Println("body", string(body))

	res.Took = time.Since(start).String()
	// fmt.Println("body", string(body))
	return &res, nil
}

// func (c *WebClient) Ack(item string) error {
// 	finalurl := fmt.Sprintf("%s/tinyq/ack?item=%s", c.url, item)

// 	_, err := c.simpleget(finalurl)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (c *WebClient) Get(key string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/crud/get/%s", c.url, key)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", notiteminqueue
	}

	return body.Message, nil
}

func (c *WebClient) Set(key, value string) error {
	finalurl := fmt.Sprintf("%s/tinyq/crud/set/%s?v=%s", c.url, key, value)
	_, err := c.simpleget(finalurl)
	if err != nil {
		return err
	}

	return nil
}

func (c *WebClient) Delete(key string) error {
	finalurl := fmt.Sprintf("%s/tinyq/crud/delete/%s", c.url, key)
	_, err := c.simpleget(finalurl)
	if err != nil {
		return err
	}

	return nil
}

func (c *WebClient) Pop(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/pop?channel=%s", c.url, channel)
	// fmt.Println("finalurl", finalurl)
	body, err := c.simpleget(finalurl)
	if err != nil {
		// fmt.Println("error in pop", err)
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", notiteminqueue
	}

	// if err = c.Ack(string(body)); err != nil {
	// 	return empty, err
	// }

	return body.Message, nil
}

func (c *WebClient) Channels() (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels", c.url)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", nil
	}

	return body.Message, nil
}

func (c *WebClient) LockChannel(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/lock?channel=%s", c.url, channel)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", nil
	}

	return body.Message, nil
}

func (c *WebClient) UnlockChannel(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/unlock?channel=%s", c.url, channel)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", nil
	}

	return body.Message, nil
}

func (c *WebClient) ChannelLockStatus(channel string) (bool, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/lockstatus?channel=%s", c.url, channel)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return false, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return false, nil
	}

	return body.Message == "locked", nil
}

func (c *WebClient) PauseChannel(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/pause?channel=%s", c.url, channel)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", nil
	}

	return body.Message, nil
}

func (c *WebClient) ClearChannel(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/clear?channel=%s", c.url, channel)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", nil
	}

	return body.Message, nil
}

func (c *WebClient) DeleteChannel(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/delete?channel=%s", c.url, channel)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", nil
	}

	return body.Message, nil
}

func (c *WebClient) ResumeChannel(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/resume?channel=%s", c.url, channel)
	body, err := c.simpleget(finalurl)
	if err != nil {
		return empty, err
	}

	if strings.EqualFold(body.Message, "error") || strings.EqualFold(body.Message, "empty") || strings.EqualFold(body.Message, "paused") {
		return "", nil
	}

	return body.Message, nil
}

func (c *WebClient) Route(channel, id string, pairs ...any) error {
	if len(pairs) > 0 {
		data := serializepairs(pairs...)
		if len(data) > 0 {
			return c.Push(channel + "." + id + "." + data)
		}
	}

	return c.Push(channel + "." + id)
}

func (c *WebClient) RouteWithData(channel, id string, data string) error {
	if len(data) > 0 {
		return c.Push(channel + "." + id + "." + data)
	}
	return c.Push(channel + "." + id)
}

func (c *WebClient) RouteItem(item string) error {
	return c.Push(item)
}

func (c *WebClient) Push(item string) error {
	finalurl := fmt.Sprintf("%s/tinyq/push?item=%s", c.url, item)

	_, err := c.simpleget(finalurl)
	if err != nil {
		return err
	}

	return nil
}

// func (c *WebClient) Pause() error {
// 	finalurl := fmt.Sprintf("%s/tinyq/channels/pause", c.url)

// 	if _, err := c.simpleget(finalurl); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (c *WebClient) Resume() error {
// 	finalurl := fmt.Sprintf("%s/tinyq/channels/resume", c.url)

// 	if _, err := c.simpleget(finalurl); err != nil {
// 		return err
// 	}

// 	return nil
// }

func (c *WebClient) PauseStatus(channel string) (string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/channels/status?channel=%s", c.url, channel)

	body, err := c.simpleget(finalurl)
	if err != nil {
		return "", err
	}

	return body.Message, nil
}

func (c *WebClient) Databases() ([]string, error) {
	finalurl := fmt.Sprintf("%s/tinyq/databases", c.url)

	body, err := c.simpleget(finalurl)
	if err != nil {
		return nil, err
	}

	var i []string
	err = json.Unmarshal([]byte(body.Message), &i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (c *WebClient) Stats() ([]*tinyq.ChannelStats, error) {
	finalurl := fmt.Sprintf("%s/tinyq/stats", c.url)

	body, err := c.simpleget(finalurl)
	if err != nil {
		return nil, err
	}

	var i []*tinyq.ChannelStats
	err = json.Unmarshal([]byte(body.Message), &i)
	if err != nil {
		fmt.Println("stats", body)
		return nil, err
	}

	return i, nil
}

func (c *WebClient) WorkerLoop(channel string, callback TqWorker) {
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
				if err == notiteminqueue {
					err = nil
					time.Sleep(*c.backoffduration)
				}
				// fmt.Println("error popping item:", err)
				continue
			}

			item = strings.TrimSpace(item)

			if item == empty || item == "error" || item == "empty" || item == "paused" {
				// fmt.Println("sleep: backoff - " + channel)
				time.Sleep(*c.backoffduration)
				continue
			}

			_, key, data := tinyq.Splititem(item)

			ctx := WebWorkerContext{
				Item:    item,
				ID:      key,
				Client:  c,
				Channel: channel,
			}

			if data != "" {
				m := deserialize(data)
				if len(m) > 0 {
					ctx.Data = m
				}
			}

			func() {
				defer func() {
					if err := recover(); err != nil {
						fmt.Println("panic recovered:", err)
					}
				}()

				next := callback(&ctx)
				// fmt.Println("Next", next)
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
