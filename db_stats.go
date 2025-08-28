package tinyq

import (
	"strconv"
	"strings"

	"go.etcd.io/bbolt"
)

type ChannelStats struct {
	Channel  string         `json:"channel"`
	Stats    map[string]int `json:"stats"`
	IsPaused bool           `json:"is_paused"`
	Count    int            `json:"count"`
}

func (s *tinyQ) Stats(appname string) ([]*ChannelStats, error) {
	stats := make(map[string]int)

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(appname))
		if bucket == nil {
			return nil //no stats yet
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			key := string(k)
			value := string(v)
			stats[key], _ = strconv.Atoi(string(value))
		}

		return nil
	})

	channels, _ := s.ListChannels()
	var statsmap []*ChannelStats
	for ch, count := range channels {

		ispaused, _ := s.IsChannelPaused(ch)
		var one = &ChannelStats{Stats: make(map[string]int), Count: count, Channel: ch, IsPaused: ispaused}

		for k, sm := range stats {
			if strings.Contains(k, ch) {
				one.Stats[strings.Replace(k, "."+ch, "", 1)] = sm
			}
		}

		statsmap = append(statsmap, one)
	}

	if err != nil {
		return nil, err
	}

	return statsmap, nil
}
