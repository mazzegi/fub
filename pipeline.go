package fub

import (
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

func NewPipeline(conn net.Conn) (*Pipeline, error) {
	listener, err := net.Listen("tcp", "")
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
	Infof("pipeline: wait for incomming connection ...")
	timer := time.AfterFunc(10*time.Second, func() {
		Warnf("no connect after 10 seconds - close")
		p.listener.Close()
	})
	conn, err := p.listener.Accept()
	if err != nil {
		Infof("pipeline: stop: no incomming connection")
		p.inConn.Close()
		return
	}
	timer.Stop()

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

	defer close(p.doneC)
	for {
		select {
		case <-p.stopC:
			conn.Close()
			p.inConn.Close()
			<-outInDoneC
			<-inOutDoneC
			return
		case <-outInDoneC:
			conn.Close()
			p.inConn.Close()
			<-inOutDoneC
			return
		case <-inOutDoneC:
			conn.Close()
			p.inConn.Close()
			<-outInDoneC
			return
		}
	}
}
