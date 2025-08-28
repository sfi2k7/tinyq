package server

import "sync"

type queueServer struct {
	port      int
	logging   bool
	rootpath  string
	isrunning bool
	qm        *queuemanager
	sm        *statemanager
	admin     *admin
}

type Option func(*queueServer)

func WithPort(port int) Option {
	return func(s *queueServer) {
		s.port = port
	}
}

func WithLogging(logging bool) Option {
	return func(s *queueServer) {
		s.logging = logging
	}
}

func WithRootPath(rootpath string) Option {
	return func(s *queueServer) {
		s.rootpath = rootpath
	}
}

func NewQueueServer(options ...Option) *queueServer {
	s := &queueServer{
		port:    8080,
		logging: false,
		qm:      &queuemanager{lock: sync.Mutex{}},
		sm:      &statemanager{ch: make(chan string, 100)},
	}

	s.admin = newadmin(s.qm)
	s.sm.qm = s.qm

	for _, option := range options {
		option(s)
	}

	return s
}

func (s *queueServer) Start() error {
	if s.isrunning {
		return nil
	}

	go s.sm.Start()

	defer close(s.sm.ch)

	return s.serve()
}

func (s *queueServer) Stop() error {
	return nil
}
