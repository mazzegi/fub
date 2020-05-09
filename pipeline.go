package fub

import (
	"fmt"
	"io"
	"net"
	"time"
)

type Pipeline struct {
	inConn   net.Conn
	listener net.Listener
	stopC    chan struct{}
	doneC    chan struct{}
}

func NewPipeline(conn net.Conn, host string) (*Pipeline, error) {
	bind := fmt.Sprintf("%s:0", host)
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}
	p := &Pipeline{
		inConn:   conn,
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		listener: listener,
	}
	return p, nil
}

func (p *Pipeline) Addr() string {
	return p.listener.Addr().String()
}

func (p *Pipeline) Close() {
	close(p.stopC)
	<-p.doneC
}

func (p *Pipeline) Run() {
	defer close(p.doneC)

	connC := make(chan net.Conn)
	go func() {
		conn, err := p.listener.Accept()
		if err != nil {
			Infof("pipeline: accept-error: %v", err)
			close(connC)
			return
		}
		connC <- conn
	}()

	var conn net.Conn
	select {
	case <-p.stopC:
		p.listener.Close()
		p.inConn.Close()
		return
	case <-time.After(10 * time.Second):
		p.listener.Close()
		p.inConn.Close()
		return
	case lconn, ok := <-connC:
		if !ok {
			close(connC)
			p.listener.Close()
			p.inConn.Close()
			return
		}
		conn = lconn
	}

	Infof("pipeline: start copying")
	defer Infof("pipeline: stop copying")

	outInDoneC := make(chan struct{})
	inOutDoneC := make(chan struct{})
	go func() {
		defer close(outInDoneC)
		Infof("pipeline: start copy: in->out")
		_, err := io.Copy(conn, p.inConn)
		Infof("pipeline: copy: in->out ended: %v", err)
	}()
	go func() {
		defer close(inOutDoneC)
		Infof("pipeline: start copy: out->in")
		_, err := io.Copy(p.inConn, conn)
		Infof("pipeline: copy: out->in ended: %v", err)
	}()

	defer func() {
		conn.Close()
		p.inConn.Close()
		<-outInDoneC
		<-inOutDoneC
	}()

	for {
		select {
		case <-p.stopC:
			return
		case <-outInDoneC:
			return
		case <-inOutDoneC:
			return
		}
	}
}
