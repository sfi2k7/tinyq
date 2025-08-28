package tinyq

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"go.etcd.io/bbolt"
)

var Rootpath string
var ConfigPath string

const (
	bucketPauseStatus = "internal:pause_status"
	bucketStats       = "internal:stats"
)

var defaultOptions = &Options{
	Appname: "default",
}

func init() {
	if runtime.GOOS == "darwin" {
		Rootpath = "./"
		ConfigPath = "./"
	} else {
		Rootpath = "/var/lib/tinyq/"
		ConfigPath = "/etc/tinyq/"

		if err := createifnotexists(Rootpath); err != nil {
			panic(err)
		}

		if err := createifnotexists(ConfigPath); err != nil {
			panic(err)
		}
	}
}

type Options struct {
	Appname  string
	Rootpath string
}

type tinyQ struct {
	db     *bbolt.DB
	isOpen bool
	opt    *Options
	lock   sync.Mutex
}

func NewTinyQ(opt *Options) TinyQ {
	if opt == nil {
		opt = defaultOptions
	}

	return &tinyQ{
		isOpen: false,
		lock:   sync.Mutex{},
		opt:    opt,
	}
}

func isInternalChannel(channel string) bool {
	return strings.HasPrefix(channel, "internal:")
}

func (s *tinyQ) Open() error {
	// s.lock.Lock()
	// defer s.lock.Unlock()

	if s.isOpen {
		return nil
	}

	fmt.Println("options", s.opt)
	var err error
	var dbpath = filepath.Join(Rootpath, s.opt.Appname+".db")
	s.db, err = bbolt.Open(dbpath, 0600, nil)
	if err != nil {
		return err
	}

	s.isOpen = true
	return nil
}

func (s *tinyQ) Close() error {
	if s.db != nil {
		err := s.db.Close()
		if err != nil {
			return err
		}

		s.isOpen = false
	}

	return nil
}

func (s *tinyQ) Set(b, k, v string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(b))
		if err != nil {
			return err
		}

		return bucket.Put([]byte(k), []byte(v))
	})
}

func (s *tinyQ) Delete(b, k string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(b))
		if err != nil {
			return err
		}

		return bucket.Delete([]byte(k))
	})
}

func (s *tinyQ) Get(b, k string) (string, error) {
	var value []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(b))
		if bucket == nil {
			return errors.New("bucket not found")
		}

		value = bucket.Get([]byte(k))
		return nil
	})

	if err != nil {
		return "", err
	}

	if value == nil {
		return "", nil
	}

	return string(value), nil
}

func (s *tinyQ) Has(b, k string) (bool, error) {
	var exists bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(b))
		if bucket == nil {
			return nil
		}

		exists = bucket.Get([]byte(k)) != nil
		return nil
	})

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *tinyQ) Push(item string) error {
	channel, key, data := Splititem(item)

	err := s.db.Update(func(tx *bbolt.Tx) error {

		bucket, err := tx.CreateBucketIfNotExists([]byte(channel))
		if err != nil {
			return err
		}

		var databytes = []byte("")
		if data != "" {
			databytes = []byte(data)
		}

		return bucket.Put([]byte(key), databytes)
	})

	if err != nil {
		return err
	}

	// s.statschannel <- "push." + channel

	return nil
}

// Pop retrieves and removes one or more items from the front of the queue for a given channel.
func (s *tinyQ) Pop(channel string, count ...int) ([]string, error) {

	// Determine how many items to pop, applying a sensible default and a maximum limit.
	popCount := 1
	if len(count) > 0 && count[0] > 0 {
		popCount = count[0]
	}

	if popCount > 10 {
		popCount = 10
	}

	// Pre-allocate the slice with the desired capacity for better performance.
	items := make([]string, 0, popCount)

	err := s.db.Update(func(tx *bbolt.Tx) error {
		// Attempt to get the bucket. If it doesn't exist, the queue is empty.
		// There's no need to create it during a pop operation.
		b := tx.Bucket([]byte(channel))
		if b == nil {
			return nil // Not an error, just an empty queue.
		}

		c := b.Cursor()

		// **Corrected Loop Logic**:
		// Initialize with c.First() before the loop.
		// Iterate with c.Next() as the post-statement.
		// This correctly walks through the items instead of repeatedly getting the first one.
		for k, v := c.First(); k != nil && len(items) < popCount; k, v = c.Next() {
			// Use strings.Builder for efficient string construction.
			var sb strings.Builder
			sb.WriteString(channel)
			sb.WriteByte('.')
			sb.Write(k)
			if len(v) > 0 {
				sb.WriteByte('.')
				sb.Write(v)
			}
			items = append(items, sb.String())

			// Delete the key that was just retrieved.
			if err := b.Delete(k); err != nil {
				// If deletion fails, abort the transaction.
				return fmt.Errorf("failed to delete item %s: %w", string(k), err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err // Return any error from the database transaction.
	}

	// Update stats only if items were actually popped.
	if len(items) > 0 {
		// Note: If this secondary operation fails, the items have already been
		// successfully removed from the DB. Consider logging this error
		// instead of returning it to the caller to avoid confusion.

		// s.statschannel <- "pop." + channel
	}

	return items, nil
}

func (s *tinyQ) RemoveItem(item string) error {
	var channel string
	var key string

	if strings.Contains(item, ".") {
		channel, key, _ = Splititem(item)
	} else {
		channel = "TEST_CHANNEL"
		key = item
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channel))
		if bucket == nil {
			return errors.New("channel not found")
		}

		return bucket.Delete([]byte(key))
	})
}

func (s *tinyQ) ListAllKeys(channel string) ([]string, error) {
	var items []string
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channel))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			items = append(items, channel+"."+string(k))
			return nil
		})
	})

	return items, err
}

func (s *tinyQ) Count(channel string) (int, error) {

	var count int
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channel))
		if bucket == nil {
			return nil
		}
		count = bucket.Stats().KeyN
		return nil
	})

	return count, err
}
