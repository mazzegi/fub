package fub

import (
	"io"
	"net"
)

type Wire struct {
	fubAddr string
	locAddr string
}

func NewWire(fubAddr, locAddr string) *Wire {
	return &Wire{
		fubAddr: fubAddr,
		locAddr: locAddr,
	}
}

func (w *Wire) Run() {
	locConn, err := net.Dial("tcp", w.locAddr)
	if err != nil {
		Errorf("wire: connect to local addr %q failed: %v", w.locAddr, err)
		return
	}
	defer locConn.Close()
	fubConn, err := net.Dial("tcp", w.fubAddr)
	if err != nil {
		Errorf("wire: connect to fub addr %q failed: %v", w.fubAddr, err)
		return
	}
	defer fubConn.Close()

	Infof("wire: start copy ...")

	outInDoneC := make(chan struct{})
	inOutDoneC := make(chan struct{})
	go func() {
		defer close(outInDoneC)
		Infof("wire: start copy: in->out")
		_, err := io.Copy(locConn, fubConn)
		Infof("wire: copy: in->out ended: %v", err)
	}()
	go func() {
		defer close(inOutDoneC)
		Infof("wire: start copy: out->in")
		_, err := io.Copy(fubConn, locConn)
		Infof("wire: copy: out->in ended: %v", err)
	}()

	<-outInDoneC
	<-inOutDoneC
	Infof("wire: copy ... done")
}
