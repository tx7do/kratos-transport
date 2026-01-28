package broker

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"
	_ "github.com/go-kratos/kratos/v2/encoding/proto"
)

// Marshal encodes a message into bytes using the provided codec.
func Marshal(codec encoding.Codec, msg any) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}

	if codec != nil {
		return codec.Marshal(msg)
	}

	switch t := msg.(type) {
	case []byte:
		return t, nil
	case string:
		return []byte(t), nil
	default:
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(msg); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}

// Unmarshal decodes bytes into a message using the provided codec.
func Unmarshal(codec encoding.Codec, inputData []byte, outValue any) error {
	if inputData == nil {
		return errors.New("inputData is nil")
	}
	if outValue == nil {
		return errors.New("outValue is nil; must be a pointer to the target value")
	}

	if codec != nil {
		return codec.Unmarshal(inputData, outValue)
	}

	// No codec: support common target pointer types, otherwise use gob decode.
	switch v := outValue.(type) {
	case *[]byte:
		*v = make([]byte, len(inputData))
		copy(*v, inputData)
		return nil
	case *string:
		*v = string(inputData)
		return nil
	default:
		buf := bytes.NewBuffer(inputData)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(outValue); err != nil {
			return err
		}
		return nil
	}
}
