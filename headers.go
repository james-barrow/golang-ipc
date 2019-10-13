package ipc

import (
	"bytes"
	"encoding/binary"
)

func createHeader(version byte, messType uint32, multiPart bool, multiPartID uint32) []byte {

	var header []byte

	if multiPart == false {
		header = make([]byte, 6)
	} else {
		header = make([]byte, 10)
	}

	header[0] = byte(version) // version 1 byte

	binary.BigEndian.PutUint32(header[1:5], uint32(messType)) // message type 4 bytes

	if multiPart == false {
		header[5] = byte(0) // are we sending a multipart message 1 byte
	} else {
		header[5] = byte(1)                                           // is it is a multipart - header[5] = 1
		binary.BigEndian.PutUint32(header[6:10], uint32(multiPartID)) //
	}

	return header
}

func readHeader(header []byte) *Message {

	m := Message{}

	m.version = header[0] //- version

	binary.Read(bytes.NewReader(header[1:5]), binary.BigEndian, &m.msgType) // message type

	m.multiPart = header[5] // multipart message

	if m.multiPart != 0 {

		binary.Read(bytes.NewReader(header[6:10]), binary.BigEndian, &m.multiPartID) // multi message ID is required

	}

	return &m

}
