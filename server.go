package fub

import (
	"net"
)

type Option func(s *Server) error

func Bind(bind string) Option {
	return func(s *Server) error {
		s.bind = bind
		return nil
	}
}

type Server struct {
	bind     string
	host     string
	listener net.Listener
	stopC    chan struct{}
	doneC    chan struct{}
}

func NewServer(opts ...Option) (*Server, error) {
	s := &Server{
		bind:  "127.0.0.1:9201",
		stopC: make(chan struct{}),
		doneC: make(chan struct{}),
	}
	var err error
	for _, opt := range opts {
		err = opt(s)
		if err != nil {
			return nil, err
		}
	}
	host, _, err := net.SplitHostPort(s.bind)
	if err != nil {
		return nil, err
	}
	s.host = host

	s.listener, err = net.Listen("tcp", s.bind)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) Run() {
	defer close(s.doneC)

	channels := NewCloserCache()
	listenDoneC := make(chan struct{})
	go func() {
		defer close(listenDoneC)
		Infof("accept connections ...")
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				Infof("listener: stop")
				return
			}
			Infof("new connection from %q", conn.RemoteAddr())
			go func() {
				c := NewChannel(conn, s.host)
				id := channels.Add(c)
				c.Run()
				channels.Remove(id)
			}()
		}
	}()

	for {
		select {
		case <-s.stopC:
			Infof("received stop signal - close listener")
			s.listener.Close()
			<-listenDoneC
			channels.CloseAll()
			return
		}
	}
}

func (s *Server) Close() {
	Infof("server close ...")
	close(s.stopC)
	<-s.doneC
	Infof("server close ... done")
}
