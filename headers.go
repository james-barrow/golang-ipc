package ipc

import (
	"bytes"
	"encoding/binary"
)

func createHeader(version byte, messType uint32, messLen int) []byte {

	header := make([]byte, 9)

	header[0] = byte(version) // version 1 byte

	binary.BigEndian.PutUint32(header[1:5], uint32(messType)) // message type 4 bytes

	binary.BigEndian.PutUint32(header[5:9], uint32(messLen))

	return header
}

func processHeader(header []byte) *Message {

	m := Message{}

	m.version = header[0] //- version

	binary.Read(bytes.NewReader(header[1:5]), binary.BigEndian, &m.msgType) // message type

	var mlen uint32

	binary.Read(bytes.NewReader(header[5:9]), binary.BigEndian, &mlen) // multi message ID is required

	m.msgLen = int(mlen)

	return &m

}
