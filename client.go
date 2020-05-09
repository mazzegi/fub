package fub

import (
	"bufio"
	"net"
)

type Client struct {
	host  string
	stopC chan struct{}
	doneC chan struct{}
}

func NewClient(host string) *Client {
	c := &Client{
		host:  host,
		stopC: make(chan struct{}),
		doneC: make(chan struct{}),
	}
	return c
}

func (c *Client) Close() {
	close(c.stopC)
	<-c.doneC
}

func (c *Client) Run() {
	defer close(c.doneC)

	conn, err := net.Dial("tcp", c.host)
	if err != nil {
		Errorf("dial %s: %v", c.host, err)
		return
	}
	readC := make(chan Message)
	go func() {
		defer close(readC)
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			m, err := DecodeMessage(scanner.Bytes())
			if err != nil {
				Errorf("decode message: %v", err)
				return
			}
			readC <- m
		}
		if scanner.Err() != nil {
			Errorf("scanner finished with: %v", scanner.Err())
		} else {
			Infof("scanner finished gracefully")
		}
	}()

	for {
		select {
		case <-c.stopC:
			conn.Close()
			<-readC
			return
		case m, ok := <-readC:
			if !ok {
				return
			}
			Infof("received: %T", m)
			switch m := m.(type) {
			case InitRequest:
				WriteMessage(conn, InitResponse{
					Name: "Foo",
					Port: 4711,
				})
			case WireTo:
				Infof("received: wire-to: %s", m.Addr)
				w := NewWire(m.Addr, "127.0.0.1:5006")
				go w.Run()
			}
		}
	}
}
