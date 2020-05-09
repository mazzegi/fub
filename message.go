package fub

import (
	"io"

	"github.com/pkg/errors"
)

func WriteMessage(w io.Writer, m Message) error {
	bs, err := EncodeMessage(m)
	if err != nil {
		return errors.Errorf("encode message: %v", err)
	}
	_, err = w.Write(append(bs, '\n'))
	return err
}

type Message interface{}

type InitRequest struct{}

type InitResponse struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

type WireTo struct {
	Addr string `json:"addr"`
}
