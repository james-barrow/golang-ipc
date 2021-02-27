package ipc

import (
	"bufio"
	"errors"
	"strings"
	"time"
)

// StartClient - start the ipc client.
//
// ipcName = is the name of the unix socket or named pipe that the client will try and connect to.
// timeout = number of seconds before the socket/pipe times out trying to connect/re-cconnect - if -1 or 0 it never times out.
// retryTimer = number of seconds before the client tries to connect again.
//
func StartClient(ipcName string, config *ClientConfig) (*Client, error) {

	err := checkIpcName(ipcName)
	if err != nil {
		return nil, err

	}

	cc := &Client{
		Name:     ipcName,
		status:   NotConnected,
		recieved: make(chan *Message),
		toWrite:  make(chan *Message),
	}

	if config == nil {

		cc.timeout = 0
		cc.retryTimer = time.Duration(1)
		cc.encryptionReq = true

	} else {

		if config.Timeout < 0 {
			cc.timeout = 0
		} else {
			cc.timeout = config.Timeout
		}

		if config.RetryTimer < 1 {
			cc.retryTimer = time.Duration(1)
		} else {
			cc.retryTimer = time.Duration(config.RetryTimer)
		}

		if config.Encryption == false {
			cc.encryptionReq = false
		} else {
			cc.encryptionReq = true // defualt is to always enforce encryption
		}
	}

	go startClient(cc)

	return cc, nil

}

func startClient(cc *Client) {

	cc.status = Connecting
	cc.recieved <- &Message{Status: cc.status.String(), MsgType: -1}

	err := cc.dial()
	if err != nil {
		cc.recieved <- &Message{err: err, MsgType: -2}
		return
	}

	cc.status = Connected
	cc.recieved <- &Message{Status: cc.status.String(), MsgType: -1}

	go cc.read()
	go cc.write()

}

func (cc *Client) read() {
	bLen := make([]byte, 4)

	for {

		res := cc.readData(bLen)
		if res == false {
			break
		}

		mLen := bytesToInt(bLen)

		msgRecvd := make([]byte, mLen)

		res = cc.readData(msgRecvd)
		if res == false {
			break
		}

		if cc.encryption == true {
			msgFinal, err := decrypt(*cc.enc.cipher, msgRecvd)
			if err != nil {
				break
			}

			if bytesToInt(msgFinal[:4]) == 0 {
				//  type 0 = control message
			} else {
				cc.recieved <- &Message{Data: msgFinal[4:], MsgType: bytesToInt(msgFinal[:4])}
			}

		} else {

			if bytesToInt(msgRecvd[:4]) == 0 {
				//  type 0 = control message
			} else {
				cc.recieved <- &Message{Data: msgRecvd[4:], MsgType: bytesToInt(msgRecvd[:4])}
			}
		}
	}
}

func (cc *Client) readData(buff []byte) bool {

	_, err := cc.conn.Read(buff)
	if err != nil {
		if strings.Contains(err.Error(), "EOF") { // the connection has been closed by the client.
			cc.conn.Close()

			if cc.status != Closing || cc.status == Closed {
				go cc.reconnect()
			}
			return false
		}

		if cc.status == Closing {
			cc.status = Closed
			cc.recieved <- &Message{Status: cc.status.String(), MsgType: -1}
			cc.recieved <- &Message{err: errors.New("Client has closed the connection"), MsgType: -2}
			return false
		}

		// other read error
		return false

	}

	return true

}

func (cc *Client) reconnect() {

	cc.status = ReConnecting
	cc.recieved <- &Message{Status: cc.status.String(), MsgType: -1}

	err := cc.dial() // connect to the pipe
	if err != nil {
		if err.Error() == "Timed out trying to connect" {
			cc.status = Timeout
			cc.recieved <- &Message{Status: cc.status.String(), MsgType: -1}
			cc.recieved <- &Message{err: errors.New("Timed out trying to re-connect"), MsgType: -2}
		}

		return
	}

	cc.status = Connected
	cc.recieved <- &Message{Status: cc.status.String(), MsgType: -1}

	go cc.read()

}

// Read - blocking function that waits until an non multipart message is recieved
// returns the message type, data and any error.
//
func (cc *Client) Read() (*Message, error) {

	m, ok := (<-cc.recieved)
	if ok == false {
		return nil, errors.New("the recieve channel has been closed")
	}

	if m.err != nil {
		close(cc.recieved)
		close(cc.toWrite)
		return nil, m.err
	}

	return m, nil

}

// Write - writes a non multipart message to the ipc connection.
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
//
func (cc *Client) Write(msgType int, message []byte) error {

	if msgType == 0 {
		return errors.New("Message type 0 is reserved")
	}

	if cc.status != Connected {
		return errors.New(cc.status.String())
	}

	mlen := len(message)
	if mlen > cc.maxMsgSize {
		return errors.New("Message exceeds maximum message length")
	}

	cc.toWrite <- &Message{MsgType: msgType, Data: message}

	return nil

}

func (cc *Client) write() {

	for {

		m, ok := <-cc.toWrite

		if ok == false {
			break
		}

		toSend := intToBytes(m.MsgType)

		writer := bufio.NewWriter(cc.conn)

		if cc.encryption == true {
			toSend = append(toSend, m.Data...)
			toSendEnc, err := encrypt(*cc.enc.cipher, toSend)
			if err != nil {
				//return err
			}
			toSend = toSendEnc
		} else {

			toSend = append(toSend, m.Data...)

		}

		writer.Write(intToBytes(len(toSend)))
		writer.Write(toSend)

		err := writer.Flush()
		if err != nil {
			//return err
		}

	}
}

// getStatus - get the current status of the connection
func (cc *Client) getStatus() Status {

	return cc.status

}

// StatusCode - returns the current connection status
func (cc *Client) StatusCode() Status {
	return cc.status
}

// Status - returns the current connection status as a string
func (cc *Client) Status() string {

	return cc.status.String()

}

// Close - closes the connection
func (cc *Client) Close() {

	cc.status = Closing
	cc.conn.Close()
}
