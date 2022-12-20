package broker

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"
	_ "github.com/go-kratos/kratos/v2/encoding/proto"
)

func Marshal(codec encoding.Codec, msg Any) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}

	if codec != nil {
		dataBuffer, err := codec.Marshal(msg)
		if err != nil {
			return nil, err
		}
		return dataBuffer, nil
	} else {
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
}

func Unmarshal(codec encoding.Codec, inputData []byte, outValue interface{}) error {
	if codec != nil {
		if err := codec.Unmarshal(inputData, outValue); err != nil {
			return err
		}
	} else if outValue == nil {
		outValue = inputData
	}
	return nil
}
