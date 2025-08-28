package tinyq

import (
	"errors"

	"go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("not found")

type BoltWrapper struct {
	db     *bbolt.DB
	isopen bool
}

func NewWrapper(path string) (*BoltWrapper, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &BoltWrapper{db: db, isopen: true}, nil
}

func (w *BoltWrapper) Get(key string) (string, error) {
	var value []byte
	err := w.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("data"))
		if b == nil {
			return ErrNotFound
		}

		value = b.Get([]byte(key))
		if value == nil {
			return ErrNotFound
		}
		return nil
	})
	return string(value), err
}

func (w *BoltWrapper) Put(key, value string) error {
	return w.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("data"))
		if err != nil {
			return err
		}

		return b.Put([]byte(key), []byte(value))
	})
}

func (w *BoltWrapper) Delete(key string) error {
	return w.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("data"))
		if b == nil {
			return ErrNotFound
		}

		return b.Delete([]byte(key))
	})
}

func (w *BoltWrapper) Close() error {
	if w.db == nil {
		return nil
	}

	err := w.db.Close()
	if err != nil {
		return err
	}

	w.isopen = false
	return nil
}

func (w *BoltWrapper) ListKeys(bucket string) ([]string, error) {
	var keys []string
	err := w.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNotFound
		}

		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})
	return keys, err
}

func (w *BoltWrapper) ListValues(bucket string) ([]string, error) {
	var values []string
	err := w.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNotFound
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			values = append(values, string(v))
		}
		return nil
	})
	return values, err
}

func (w *BoltWrapper) ListKeysAndValues(bucket string) (map[string]string, error) {
	var keyvals = make(map[string]string)
	err := w.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNotFound
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			keyvals[string(k)] = string(v)
		}
		return nil
	})
	return keyvals, err
}

func (w *BoltWrapper) ReadTransection(bucket string, fn func(*bbolt.Bucket) error) error {
	return w.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNotFound
		}

		return fn(b)
	})
}

func (w *BoltWrapper) WriteTransection(bucket string, fn func(*bbolt.Bucket) error) error {
	return w.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNotFound
		}

		return fn(b)
	})
}

func (w *BoltWrapper) Count(bucket string) (int, error) {
	var count int
	err := w.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrNotFound
		}

		count = b.Stats().KeyN
		return nil
	})
	return count, err
}

func (w *BoltWrapper) ListBuckets() ([]string, error) {
	var buckets []string
	err := w.db.View(func(tx *bbolt.Tx) error {
		c := tx.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			buckets = append(buckets, string(k))
		}
		return nil
	})
	return buckets, err
}
