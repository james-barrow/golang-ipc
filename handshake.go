package ipc

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// 1st message sent from the server
// byte 0 = protocal version no.
// byte 1 = whether encryption is to be used - 0 no , 1 = encryption
func (sc *Server) handshake() error {

	err := sc.one()
	if err != nil {
		return err
	}

	if sc.encryption == true {
		err = sc.startEncryption()
		if err != nil {
			return err
		}
	}

	err = sc.msgLength()
	if err != nil {
		return err
	}

	return nil

}

func (sc *Server) one() error {

	buff := make([]byte, 2)

	buff[0] = byte(version)

	if sc.encryption == true {
		buff[1] = byte(1)
	} else {
		buff[1] = byte(0)
	}

	_, err := sc.conn.Write(buff)
	if err != nil {
		return errors.New("unable to send handshake ")
	}

	recv := make([]byte, 1)
	_, err = sc.conn.Read(recv)
	if err != nil {
		return errors.New("failed to recieve handshake reply")
	}

	switch result := recv[0]; result {
	case 0:
		return nil
	case 1:
		return errors.New("client has a different version number")
	case 2:
		return errors.New("client is enforcing encryption")
	case 3:
		return errors.New("server failed to get handshake reply")

	}

	return errors.New("other error - handshake failed")

}

func (sc *Server) startEncryption() error {

	shared, err := sc.keyExchange()
	if err != nil {
		return err
	}

	gcm, err := createCipher(shared)
	if err != nil {
		return err
	}

	sc.enc = &encryption{
		keyExchange: "ecdsa",
		encryption:  "AES-GCM-256",
		cipher:      gcm,
	}

	return nil

}

func (sc *Server) msgLength() error {

	toSend := make([]byte, 4)

	buff := make([]byte, 4)
	binary.BigEndian.PutUint32(buff, uint32(sc.maxMsgSize))

	if sc.encryption == true {
		maxMsg, err := encrypt(*sc.enc.cipher, buff)
		if err != nil {
			return err
		}

		binary.BigEndian.PutUint32(toSend, uint32(len(maxMsg)))
		toSend = append(toSend, maxMsg...)

	} else {

		binary.BigEndian.PutUint32(toSend, uint32(len(buff)))
		toSend = append(toSend, buff...)
	}

	_, err := sc.conn.Write(toSend)
	if err != nil {
		return errors.New("unable to send max message length ")
	}

	reply := make([]byte, 1)

	_, err = sc.conn.Read(reply)
	if err != nil {
		return errors.New("did not recieve message length reply")
	}

	return nil

}

// 1st message recieved by the client
func (cc *Client) handshake() error {

	err := cc.one()
	if err != nil {
		return err
	}

	if cc.encryption == true {
		err := cc.startEncryption()
		if err != nil {
			return err
		}
	}

	err = cc.msgLength()
	if err != nil {
		return err
	}

	return nil

}

func (cc *Client) one() error {

	recv := make([]byte, 2)
	_, err := cc.conn.Read(recv)
	if err != nil {
		return errors.New("failed to recieve handshake message")
	}

	if recv[0] != version {
		cc.handshakeSendReply(1)
		return errors.New("server has sent a different version number")
	}

	if recv[1] != 1 && cc.encryptionReq == true {
		cc.handshakeSendReply(2)
		return errors.New("server tried to connect without encryption")
	}

	if recv[1] == 0 {
		cc.encryption = false
	} else {
		cc.encryption = true
	}

	cc.handshakeSendReply(0) // 0 is ok
	return nil

}

func (cc *Client) startEncryption() error {

	shared, err := cc.keyExchange()

	if err != nil {
		return err
	}

	gcm, err := createCipher(shared)
	if err != nil {
		return err
	}

	cc.enc = &encryption{
		keyExchange: "ECDSA",
		encryption:  "AES-GCM-256",
		cipher:      gcm,
	}

	return nil
}

func (cc *Client) msgLength() error {

	buff := make([]byte, 4)

	_, err := cc.conn.Read(buff)
	if err != nil {
		return errors.New("failed to recieve max message length 1")
	}

	var msgLen uint32
	binary.Read(bytes.NewReader(buff), binary.BigEndian, &msgLen) // message length

	buff = make([]byte, int(msgLen))

	_, err = cc.conn.Read(buff)
	if err != nil {
		return errors.New("failed to recieve max message length 2")
	}
	var buff2 []byte
	if cc.encryption == true {
		buff2, err = decrypt(*cc.enc.cipher, buff)
		if err != nil {
			return errors.New("failed to recieve max message length 3")
		}

	} else {
		buff2 = buff
	}

	var maxMsgSize uint32
	binary.Read(bytes.NewReader(buff2), binary.BigEndian, &maxMsgSize) // message length

	cc.maxMsgSize = int(maxMsgSize)
	cc.handshakeSendReply(0)

	return nil

}

func (cc *Client) handshakeSendReply(result byte) {

	buff := make([]byte, 1)
	buff[0] = result

	cc.conn.Write(buff)

}
