package server

import (
	"fmt"
	"strings"

	"github.com/sfi2k7/tinyq"
)

type statemanager struct {
	// Define fields for the StatsManager struct
	qm *queuemanager
	ch chan string
}

func NewStateManager(qm *queuemanager) *statemanager {
	return &statemanager{
		qm: qm,
		ch: make(chan string, 100),
	}
}

func (sm *statemanager) Start() {
	q, _ := sm.qm.Get("stats")
	for msg := range sm.ch {
		if msg == "" {
			return
		}

		splitted := strings.Split(msg, "|")
		if len(splitted) != 3 {
			continue
		}

		appname := splitted[0]
		command := splitted[1]
		channel := splitted[2] //channel or item

		q.Inc(appname, command+":"+channel)
	}
}

func (sm *statemanager) AddStat(appname, command, channel string) error {
	sm.ch <- fmt.Sprintf("%s|%s|%s", appname, command, channel)
	return nil
}

func (sm *statemanager) Stats(appname string) ([]*tinyq.ChannelStats, error) {
	primaryq, _ := sm.qm.Get(appname)
	channels, err := primaryq.ListChannels()

	s, err := sm.qm.Get("states")
	if err != nil {
		return nil, err
	}

	var stats []*tinyq.ChannelStats

	for channel, count := range channels {
		ispaused, _ := s.IsChannelPaused(appname + ":" + channel)
		stats = append(stats, &tinyq.ChannelStats{
			Channel:  channel,
			Count:    count,
			IsPaused: ispaused,
		})
	}

	return stats, nil
}

func (sm *statemanager) LockChannel(appname, channel string) error {
	q, err := sm.qm.Get("states")
	if err != nil {
		return err
	}
	return q.Set("__locks__", appname+":"+channel, "locked")
}

func (sm *statemanager) UnlockChannel(appname, channel string) error {
	q, err := sm.qm.Get("states")
	if err != nil {
		return err
	}
	return q.Delete("__locks__", appname+":"+channel)
}

func (sm *statemanager) IsChannelLocked(appname, channel string) (bool, error) {
	q, err := sm.qm.Get("states")
	if err != nil {
		return false, err
	}

	v, err := q.Get("__locks__", appname+":"+channel)

	return v == "locked", err
}

func (sm *statemanager) PauseChannel(appname, channel string) error {
	q, err := sm.qm.Get("states")
	if err != nil {
		return err
	}

	return q.PauseChannel(appname + ":" + channel)
}

func (sm *statemanager) IsChannelPaused(appname, channel string) (bool, error) {
	q, err := sm.qm.Get("states")
	if err != nil {
		return false, err
	}

	return q.IsChannelPaused(appname + ":" + channel)
}

func (sm *statemanager) ResumeChannel(appname, channel string) error {
	q, err := sm.qm.Get("states")
	if err != nil {
		return err
	}

	return q.UnpauseChannel(appname + ":" + channel)
}

func (sm *statemanager) SecureApp(appname string) error {
	q, err := sm.qm.Get("states")
	if err != nil {
		return err
	}

	return q.Set("__secure__", appname+":secured", "1")
}

func (sm *statemanager) IsAppSecured(appname string) (bool, error) {
	q, err := sm.qm.Get("states")
	if err != nil {
		return false, err
	}

	v, err := q.Get("__secure__", appname+":secured")

	return v == "1", err
}

func (sm *statemanager) SetAppToken(appname, token string) error {
	q, err := sm.qm.Get("states")
	if err != nil {
		return err
	}

	return q.Set("__token__", appname+":token", token)
}

func (sm *statemanager) GetAppToken(appname string) (string, error) {
	q, err := sm.qm.Get("states")
	if err != nil {
		return "", err
	}

	v, err := q.Get("__token__", appname+":token")

	return v, err
}
