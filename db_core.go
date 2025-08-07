package tinyq

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.etcd.io/bbolt"
)

var rootpath string

func init() {
	if runtime.GOOS == "darwin" {
		rootpath = "./"
	} else {
		rootpath = "/var/lib/tinyq/"

		_, err := os.Stat(rootpath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(rootpath, 0755)
			if err != nil {
				panic("Failed to create root path: " + err.Error())
			}
		}
	}
}

type TinyQ interface {
	Push(item string) error
	Pop(channel string, count ...int) ([]string, error)
	ListAll(channel string) ([]string, error)
	RemoveItem(item string) error
	ListChannels() ([]string, error)
	Count(channel string) (int, error)
	Close() error
	Open() error
	Serve() error
}

type Options struct {
	Appname string
	Port    int
}

func NewTinyQ(opt Options) TinyQ {
	return &tinyQ{
		isOpen: false,
		app:    opt.Appname,
		port:   opt.Port,
	}
}

func splititem(item string) (channel, key string) {
	splitted := strings.Split(item, ".")
	if len(splitted) != 2 {
		return "", ""
	}
	return splitted[0], splitted[1]
}

type tinyQ struct {
	db     *bbolt.DB
	app    string
	isOpen bool
	port   int
}

func (s *tinyQ) Open() error {
	if s.isOpen {
		return nil
	}

	var err error
	s.db, err = bbolt.Open(filepath.Join(rootpath, s.app+".db"), 0600, nil)
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

func (s *tinyQ) Set(k, v string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("default_store"))
		if err != nil {
			return err
		}

		return bucket.Put([]byte(k), []byte(v))
	})
}

func (s *tinyQ) Get(k string) (string, error) {
	var value []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("default_store"))
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

func (s *tinyQ) Push(item string) error {
	if !s.isOpen {
		return s.Open()
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		channel, key := splititem(item)

		bucket, err := tx.CreateBucketIfNotExists([]byte(channel))
		if err != nil {
			return err
		}

		return bucket.Put([]byte(key), []byte(""))
	})
}

func (s *tinyQ) Pop(channel string, count ...int) ([]string, error) {
	co := 1
	if !s.isOpen {
		return nil, s.Open()
	}

	if len(count) > 0 {
		co = count[0]
	}

	if co > 10 {
		co = 10 // Limit to a maximum of 10 items
	}

	var items []string
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channel))
		if bucket == nil {
			return errors.New("channel not found")
		}

		c := bucket.Cursor()
		for i := 0; i < co; i++ {
			k, _ := c.First()
			if k == nil {
				break // No more items to pop
			}

			items = append(items, channel+"."+string(k))
			err := bucket.Delete(k) // Remove the item after popping
			if err != nil {
				return err
			}

			c.Next() // Move to the next item
		}
		return nil
	})

	return items, err
}

func (s *tinyQ) RemoveItem(item string) error {
	if !s.isOpen {
		return s.Open()
	}

	var channel string
	var key string

	if strings.Contains(item, ".") {
		channel, key = splititem(item)
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

func (s *tinyQ) ListAll(channel string) ([]string, error) {
	if !s.isOpen {
		return nil, s.Open()
	}

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

func (s *tinyQ) ListChannels() ([]string, error) {
	if !s.isOpen {
		return nil, s.Open()
	}

	var channels []string
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bbolt.Bucket) error {
			channels = append(channels, string(name))
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return channels, nil
}

func (s *tinyQ) Count(channel string) (int, error) {
	if !s.isOpen {
		return 0, s.Open()
	}

	var count int
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(channel))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
	})

	return count, err
}
