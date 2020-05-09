package fub

import (
	"net"
	"sync"
)

type Channels struct {
	sync.RWMutex
	channels map[*Channel]struct{}
}

func NewChannels() *Channels {
	return &Channels{
		channels: map[*Channel]struct{}{},
	}
}

func (cs *Channels) Add(c *Channel) {
	cs.Lock()
	defer cs.Unlock()
	cs.channels[c] = struct{}{}
}

func (cs *Channels) Remove(c *Channel) {
	cs.Lock()
	defer cs.Unlock()
	delete(cs.channels, c)
}

func (cs *Channels) CloseAll() {
	cs.Lock()
	defer cs.Unlock()
	for c := range cs.channels {
		c.Close()
	}
	cs.channels = map[*Channel]struct{}{}
}

type Option func(s *Server) error

func Bind(bind string) Option {
	return func(s *Server) error {
		s.bind = bind
		return nil
	}
}

type Server struct {
	bind     string
	listener net.Listener
	stopC    chan struct{}
	doneC    chan struct{}
	channels *Channels
}

func NewServer(opts ...Option) (*Server, error) {
	s := &Server{
		bind:     ":9201",
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		channels: NewChannels(),
	}
	var err error
	for _, opt := range opts {
		err = opt(s)
		if err != nil {
			return nil, err
		}
	}
	s.listener, err = net.Listen("tcp", s.bind)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) Run() {
	defer close(s.doneC)

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
				c := NewChannel(conn)
				s.channels.Add(c)
				c.Run()
				s.channels.Remove(c)
			}()
		}
	}()

	for {
		select {
		case <-s.stopC:
			Infof("received stop signal - close listener")
			s.listener.Close()
			<-listenDoneC
			s.channels.CloseAll()
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
