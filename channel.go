package fub

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
)

type Channel struct {
	host        string
	conn        net.Conn
	stopC       chan struct{}
	doneC       chan struct{}
	callTimeout time.Duration
	listener    net.Listener
}

func NewChannel(conn net.Conn, host string) *Channel {
	return &Channel{
		host:        host,
		conn:        conn,
		stopC:       make(chan struct{}),
		doneC:       make(chan struct{}),
		callTimeout: 10 * time.Second,
	}
}

func (c *Channel) Close() {
	Infof("channel: close ...")
	close(c.stopC)
	<-c.doneC
	Infof("channel: close ... done")
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
	bind := fmt.Sprintf("%s:%d", c.host, ires.Port)
	c.listener, err = net.Listen("tcp", bind)
	if err != nil {
		Errorf("listen on %q failed: %v", bind, err)
		c.ReportError(err.Error(), readC)
		c.conn.Close()
		<-readC
		return
	}

	pipelines := NewCloserCache()
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
			pl, err := NewPipeline(conn, c.host)
			if err != nil {
				Errorf("channel: new pipeline: %v", err)
			}
			addr := pl.Addr()
			Infof("channel: spawn pipeline with addr %q", addr)
			go func() {
				id := pipelines.Add(pl)
				pl.Run()
				pipelines.Remove(id)
			}()
			WriteMessage(c.conn, WireTo{Addr: addr})
			//c.RequestWire(addr, readC)
		}
	}()
	defer func() {
		Infof("channel: close listener ...")
		c.listener.Close()
		<-listenDoneC
		Infof("channel: close listener ... done.")
		Infof("channel: close pipelines ...")
		pipelines.CloseAll()
		Infof("channel: close pipelines ... done")
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

func (c *Channel) ReportError(errs string, readC chan Message) (Ack, error) {
	m, err := c.Call(ReportError{Error: errs}, readC)
	if err != nil {
		return Ack{}, err
	}
	switch m := m.(type) {
	case Ack:
		return m, nil
	default:
		return Ack{}, errors.Errorf("response to init request is not init response but %T", m)
	}
}

// func (c *Channel) RequestWire(addr string, readC chan Message) (Ack, error) {
// 	m, err := c.Call(WireTo{Addr: addr}, readC)
// 	if err != nil {
// 		return Ack{}, err
// 	}
// 	switch m := m.(type) {
// 	case Ack:
// 		return m, nil
// 	default:
// 		return Ack{}, errors.Errorf("response to init request is not init response but %T", m)
// 	}
// }
