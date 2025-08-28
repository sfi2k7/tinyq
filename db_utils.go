package tinyq

import (
	"errors"
	"strconv"

	"go.etcd.io/bbolt"
)

func (s *tinyQ) Inc(b, k string) error {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(b))
		if err != nil {
			return err
		}

		value := bucket.Get([]byte(k))
		if value == nil {
			value = []byte("0")
		}

		count, err := strconv.Atoi(string(value))
		if err != nil {
			return err
		}

		count++
		return bucket.Put([]byte(k), []byte(strconv.Itoa(count)))
	})
	return err
}

func (s *tinyQ) PauseChannel(channel string) error {
	return s.Set(bucketPauseStatus, channel, "")
}

func (s *tinyQ) UnpauseChannel(channel string) error {
	return s.Delete(bucketPauseStatus, channel)
}

func (s *tinyQ) IsChannelPaused(channel string) (bool, error) {
	exists, err := s.Has(bucketPauseStatus, channel)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *tinyQ) ClearChannel(channel string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channel))
		if bucket == nil {
			return errors.New("channel not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			return bucket.Delete(k)
		})
	})
}

func (s *tinyQ) DeleteChannel(channel string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket([]byte(channel))
	})
}

func (s *tinyQ) ListChannels() (map[string]int, error) {

	var channels = make(map[string]int)
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			if isInternalChannel(string(name)) {
				return nil
			}

			channels[string(name)] = b.Stats().KeyN
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return channels, nil
}

func (s *tinyQ) Pop_Old(channel string, count ...int) ([]string, error) {
	co := 1
	if !s.isOpen {
		err := s.Open()
		if err != nil {
			return nil, err
		}
	}

	if len(count) > 0 {
		co = count[0]
	}

	if co > 10 {
		co = 10 // Limit to a maximum of 10 items
	}

	var items []string
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(channel))
		if err != nil {
			return errors.New("error creating channel")
		}

		if bucket == nil {
			return errors.New("channel not found")
		}

		c := bucket.Cursor()
		for i := 0; i < co; i++ {
			k, v := c.First()
			if k == nil {
				break // No more items to pop
			}

			item := channel + "." + string(k)
			if len(v) > 0 {
				item = item + "." + string(v)
			}

			items = append(items, item)

			err := bucket.Delete(k) // Remove the item after popping
			if err != nil {
				return err
			}

			c.Next() // Move to the next item
		}
		return nil
	})

	if len(items) > 0 {
		err = s.Inc(bucketStats, "pop."+channel)
		if err != nil {
			return items, err
		}
	}

	return items, err
}
