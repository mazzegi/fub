package fub

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Pipelines struct {
	sync.RWMutex
	pipelines map[*Pipeline]struct{}
}

func NewPipelines() *Pipelines {
	return &Pipelines{
		pipelines: map[*Pipeline]struct{}{},
	}
}

func (ps *Pipelines) Add(p *Pipeline) {
	ps.Lock()
	defer ps.Unlock()
	ps.pipelines[p] = struct{}{}
}

func (ps *Pipelines) Remove(p *Pipeline) {
	ps.Lock()
	defer ps.Unlock()
	delete(ps.pipelines, p)
}

func (ps *Pipelines) CloseAll() {
	ps.Lock()
	defer ps.Unlock()
	for p := range ps.pipelines {
		p.Close()
	}
	ps.pipelines = map[*Pipeline]struct{}{}
}

type Channel struct {
	conn        net.Conn
	stopC       chan struct{}
	doneC       chan struct{}
	callTimeout time.Duration
	listener    net.Listener
}

func NewChannel(conn net.Conn) *Channel {
	return &Channel{
		conn:        conn,
		stopC:       make(chan struct{}),
		doneC:       make(chan struct{}),
		callTimeout: 10 * time.Second,
	}
}

func (c *Channel) Close() {
	close(c.stopC)
	<-c.doneC
}

func (c *Channel) Run() {
	defer Infof("channel: done")
	defer close(c.doneC)

	readC := make(chan Message)
	go func() {
		defer close(readC)
		scanner := bufio.NewScanner(c.conn)
		for scanner.Scan() {
			m, err := DecodeMessage(scanner.Bytes())
			if err != nil {
				Errorf("decode message: %v", err)
				return
			}
			readC <- m
		}
		if scanner.Err() != nil {
			Warnf("scanner finished with: %v", scanner.Err())
		} else {
			Infof("scanner finished gracefully")
		}
	}()

	ires, err := c.RequestInit(readC)
	if err != nil {
		Errorf("call init-request: %v", err)
		c.conn.Close()
		<-readC
		return
	}
	//spawn listener
	Infof("got init-response: name=%s port=%d ... spawn listener", ires.Name, ires.Port)
	bind := fmt.Sprintf(":%d", ires.Port)
	c.listener, err = net.Listen("tcp", bind)
	if err != nil {
		Errorf("listen on %q failed", bind)
		c.conn.Close()
		<-readC
		return
	}

	pipelines := NewPipelines()
	listenDoneC := make(chan struct{})
	go func() {
		defer close(listenDoneC)
		Infof("channel: accept connections ...")
		for {
			conn, err := c.listener.Accept()
			if err != nil {
				Infof("channel: listener: stop")
				return
			}
			Infof("channel: new connection from %q", conn.RemoteAddr())
			pl, err := NewPipeline(conn)
			if err != nil {
				Errorf("channel: new pipeline: %v", err)
			}
			addr := pl.Addr()
			Infof("channel: spawn pipeline with addr %q", addr)
			go func() {
				pipelines.Add(pl)
				pl.Run()
				pipelines.Remove(pl)
			}()
			WriteMessage(c.conn, WireTo{
				Addr: addr,
			})
		}
	}()
	defer func() {
		c.listener.Close()
		<-listenDoneC
		pipelines.CloseAll()
	}()

	for {
		select {
		case <-c.stopC:
			c.conn.Close()
			<-readC
			return
		case m, ok := <-readC:
			if !ok {
				return
			}
			Warnf("message without request: %T", m)
		}
	}
}

func (c *Channel) Call(out Message, readC chan Message) (Message, error) {
	err := WriteMessage(c.conn, out)
	if err != nil {
		return nil, err
	}
	select {
	case in := <-readC:
		return in, nil
	case <-time.After(c.callTimeout):
		return nil, errors.Errorf("timeout in call")
	}
}

func (c *Channel) RequestInit(readC chan Message) (InitResponse, error) {
	m, err := c.Call(InitRequest{}, readC)
	if err != nil {
		return InitResponse{}, err
	}
	switch m := m.(type) {
	case InitResponse:
		return m, nil
	default:
		return InitResponse{}, errors.Errorf("response to init request is not init response but %T", m)
	}
}
