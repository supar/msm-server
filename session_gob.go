package main

import (
	"bytes"
	"encoding/gob"
)

func init() {
	gob.Register([]interface{}{})
	gob.Register(map[int]interface{}{})
	gob.Register(map[string]interface{}{})
	gob.Register(map[interface{}]interface{}{})
	gob.Register(map[string]string{})
	gob.Register(map[int]string{})
	gob.Register(map[int]int{})
	gob.Register(map[int]int64{})
}

func EncodeGob(obj map[interface{}]interface{}) ([]byte, error) {
	var (
		buffer  *bytes.Buffer
		encoder *gob.Encoder
	)

	for _, v := range obj {
		gob.Register(v)
	}

	buffer = bytes.NewBuffer(nil)
	encoder = gob.NewEncoder(buffer)

	if err := encoder.Encode(obj); err != nil {
		return []byte(""), err
	}

	return buffer.Bytes(), nil
}

func DecodeGob(encoded []byte) (out map[interface{}]interface{}, err error) {
	var (
		buffer  *bytes.Buffer
		decoder *gob.Decoder
	)

	buffer = bytes.NewBuffer(encoded)
	decoder = gob.NewDecoder(buffer)

	if err = decoder.Decode(&out); err != nil {
		return nil, err
	}

	return
}
