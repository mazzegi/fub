package fub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

var messageTypeRegistry *MessageTypeRegistry

func init() {
	messageTypeRegistry = NewMessageTypeRegistry()
	messageTypeRegistry.Register("init-request", InitRequest{})
	messageTypeRegistry.Register("init-response", InitResponse{})
	messageTypeRegistry.Register("wire-to", WireTo{})
}

type MessageTypeRegistry struct {
	types map[string]Message
}

func NewMessageTypeRegistry() *MessageTypeRegistry {
	return &MessageTypeRegistry{
		types: map[string]Message{},
	}
}

func DecodeMessage(b []byte) (Message, error) {
	return messageTypeRegistry.Decode(b)
}

func EncodeMessage(m Message) ([]byte, error) {
	return messageTypeRegistry.Encode(m)
}

func (r *MessageTypeRegistry) Register(name string, prototype Message) {
	r.types[name] = prototype
}

func (r *MessageTypeRegistry) Decode(b []byte) (Message, error) {
	i := bytes.Index(b, []byte("|"))
	if i < 0 {
		return nil, errors.Errorf("invalid message-format. missing type delimiter |")
	}
	ty := string(b[:i])
	data := b[i+1:]
	m, ok := r.types[ty]
	if !ok {
		return nil, errors.Errorf("unknown message type %q", ty)
	}
	pointerToI := reflect.New(reflect.TypeOf(m))
	err := json.Unmarshal(data, pointerToI.Interface())
	if err != nil {
		return nil, err
	}
	return pointerToI.Elem().Interface().(Message), nil
}

func (r *MessageTypeRegistry) findTypeName(m Message) (string, bool) {
	for typeName, proto := range r.types {
		if reflect.TypeOf(m) == reflect.TypeOf(proto) {
			return typeName, true
		}
	}
	return "", false
}

func (r *MessageTypeRegistry) Encode(m Message) ([]byte, error) {
	typeName, contains := r.findTypeName(m)
	if !contains {
		return nil, fmt.Errorf("encode: (%s) is not registered", reflect.TypeOf(m).Name())
	}
	mdata, err := json.Marshal(m)
	if err != nil {
		return nil, errors.Wrap(err, "marshal json")
	}
	data := []byte(typeName + "|")
	data = append(data, mdata...)
	return data, nil
}
