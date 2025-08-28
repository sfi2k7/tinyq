package server

import (
	"errors"
	"sync"

	"github.com/sfi2k7/tinyq"
)

type queuemanager struct {
	queues sync.Map
	lock   sync.Mutex
}

func (qm *queuemanager) Get(name string) (tinyq.TinyQ, error) {
	err := qm.ensureQueue(name)

	if err != nil {
		return nil, err
	}

	tq, ok := qm.queues.Load(name)
	if !ok {
		return nil, errors.New("queue not found")
	}

	return tq.(tinyq.TinyQ), nil
}

func (qm *queuemanager) hasQueue(name string) bool {
	_, ok := qm.queues.Load(name)
	return ok
}

func (qm *queuemanager) ensureQueue(name string) error {
	if qm.hasQueue(name) {
		return nil
	}

	return qm.loadQueue(name)
}

func (qm *queuemanager) loadQueue(name string) error {
	qm.lock.Lock()
	defer qm.lock.Unlock()

	tq := tinyq.NewTinyQ(&tinyq.Options{
		Appname: name,
	})

	err := tq.Open()
	if err != nil {
		return err
	}

	qm.queues.Store(name, tq)

	return nil
}

func (qm *queuemanager) Detach(name string) error {
	if !qm.hasQueue(name) {
		return errors.New("queue not found")
	}

	tq, ok := qm.queues.Load(name)
	if !ok {
		return errors.New("queue not found")
	}

	err := tq.(tinyq.TinyQ).Close()
	if err != nil {
		return err
	}

	qm.queues.Delete(name)

	return nil
}
