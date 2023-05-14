package ipc

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"time"
)

// StartClient - start the ipc client.
//
// ipcName = is the name of the unix socket or named pipe that the client will try and connect to.
// timeout = number of seconds before the socket/pipe times out trying to connect/re-cconnect - if -1 or 0 it never times out.
// retryTimer = number of seconds before the client tries to connect again.
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

		if !config.Encryption {
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
		if !res {
			break
		}

		mLen := bytesToInt(bLen)

		msgRecvd := make([]byte, mLen)

		res = cc.readData(msgRecvd)
		if !res {
			break
		}

		if cc.encryption {
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

	_, err := io.ReadFull(cc.conn, buff)
	//_, err := cc.conn.Read(buff)
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

func (c *Client) reconnect() {

	c.status = ReConnecting
	c.recieved <- &Message{Status: c.status.String(), MsgType: -1}

	err := c.dial() // connect to the pipe
	if err != nil {
		if err.Error() == "Timed out trying to connect" {
			c.status = Timeout
			c.recieved <- &Message{Status: c.status.String(), MsgType: -1}
			c.recieved <- &Message{err: errors.New("timed out trying to re-connect"), MsgType: -2}
		}

		return
	}

	c.status = Connected
	c.recieved <- &Message{Status: c.status.String(), MsgType: -1}

	go c.read()
}

// Read - blocking function that waits until an non multipart message is recieved
// returns the message type, data and any error.
func (c *Client) Read() (*Message, error) {

	m, ok := (<-c.recieved)
	if !ok {
		return nil, errors.New("the recieve channel has been closed")
	}

	if m.err != nil {
		close(c.recieved)
		close(c.toWrite)
		return nil, m.err
	}

	return m, nil
}

// Write - writes a non multipart message to the ipc connection.
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
func (c *Client) Write(msgType int, message []byte) error {

	if msgType == 0 {
		return errors.New("Message type 0 is reserved")
	}

	if c.status != Connected {
		return errors.New(c.status.String())
	}

	mlen := len(message)
	if mlen > c.maxMsgSize {
		return errors.New("Message exceeds maximum message length")
	}

	c.toWrite <- &Message{MsgType: msgType, Data: message}

	return nil
}

func (c *Client) write() {

	for {

		m, ok := <-c.toWrite

		if !ok{
			break
		}

		toSend := intToBytes(m.MsgType)

		writer := bufio.NewWriter(c.conn)

		if c.encryption {
			toSend = append(toSend, m.Data...)
			toSendEnc, err := encrypt(*c.enc.cipher, toSend)
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
func (c *Client) getStatus() Status {

	return c.status
}

// StatusCode - returns the current connection status
func (c *Client) StatusCode() Status {
	return c.status
}

// Status - returns the current connection status as a string
func (c *Client) Status() string {

	return c.status.String()
}

// Close - closes the connection
func (c *Client) Close() {

	c.status = Closing

	if c.conn != nil {
		c.conn.Close()
	}
}
