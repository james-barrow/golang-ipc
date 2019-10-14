package ipc

import (
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
		recieved: make(chan Message),
	}

	if config == nil {

		cc.timeout = 0
		cc.retryTimer = time.Duration(2)
		cc.maxMsgSize = maxMsgSize

	} else {

		if config.Timeout < 0 {
			cc.timeout = 0
		} else {
			cc.timeout = config.Timeout
		}

		if config.RetryTimer < 1 {
			cc.retryTimer = time.Duration(2)
		} else {
			cc.retryTimer = time.Duration(config.RetryTimer)
		}

		if config.MaxMsgSize < 1024 {
			cc.maxMsgSize = maxMsgSize
		} else {
			cc.maxMsgSize = config.MaxMsgSize
		}
	}

	go startClient(cc)

	return cc, nil

}

func startClient(cc *Client) {

	cc.status = Connecting

	err := cc.dial()
	if err != nil {
		cc.recieved <- Message{err: err, msgType: 0}
		return
	}

	cc.status = Connected

	go cc.read()

}

func (cc *Client) read() {

	buff := make([]byte, cc.maxMsgSize+14)
	for {

		i, err := cc.conn.Read(buff)
		if err != nil {
			if strings.Contains(err.Error(), "EOF") { // the connection has been closed by the client.
				cc.conn.Close()

				if cc.status != Closing || cc.status == Closed {
					go cc.reconnect()
				}
				break

			} else { // another error has occoured

				if cc.status == Closing {
					cc.recieved <- Message{err: errors.New("Connection closed"), msgType: 0}
				} //else {
				//cc.recieved <- Message{err: errors.New("Server closed the connection"), msgType: 0}
				//}

				cc.status = Closed
				cc.conn.Close()
				close(cc.recieved)
				break
			}
		}

		header := readHeader(buff[:10])

		if header.msgType == 0 {
			cc.msgTypeZero(buff[6:i])
		} else {

			if header.version != version {
				cc.writeControlMessage("WV")
				cc.conn.Close()
				cc.recieved <- Message{err: errors.New("Server sent the wrong version number"), msgType: 0}
				break
			}

			if header.multiPart == 0 {
				cc.recieved <- Message{data: buff[6:i], msgType: header.msgType, multiPart: header.multiPart, multiPartID: header.multiPartID}
			} else {
				// might want to do something different with multipart messages
				cc.recieved <- Message{data: buff[10:i], msgType: header.msgType, multiPart: header.multiPart, multiPartID: header.multiPartID}
			}

		}
	}
}

func (cc *Client) msgTypeZero(buf []byte) {

	// Wrong version error.
	if string(buf) == "WV" {
		cc.recieved <- Message{err: errors.New("Server recieved wrong version number"), msgType: 0}
	}

}

func (cc *Client) writeControlMessage(mess string) {

	message := []byte(mess)

	header := createHeader(version, 0, false, 0)
	header = append(header, message...)

	_, _ = cc.conn.Write(header)
}

func (cc *Client) reconnect() {

	cc.status = ReConnecting

	err := cc.dial() // connect to the pipe
	if err != nil {
		if err.Error() == "Timed out trying to connect" {
			cc.recieved <- Message{err: errors.New("Timed out trying to re-connect"), msgType: 0}
		}
		close(cc.recieved)
		return
	}

	cc.status = Connected

	go cc.read()

}

// Read - blocking function that waits until an non multipart message is recieved
// returns the message type, data and any error.
//
func (cc *Client) Read() (uint32, []byte, error) {

	m, ok := (<-cc.recieved)
	if ok == false {
		return 0, nil, errors.New("the recieve channel has been closed")
	}

	return m.msgType, m.data, m.err

}

// Write - writes a non multipart message to the ipc connection.
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
//
func (cc *Client) Write(msgType uint32, message []byte) error {

	if msgType == 0 {
		return errors.New("Message type 0 is reserved")
	}

	if len(message) > cc.maxMsgSize {
		return errors.New("Message exceeds maximum message length")
	}

	if cc.status == Connected {

		header := createHeader(version, msgType, false, 34544)
		header = append(header, message...)

		_, _ = cc.conn.Write(header)

	} else {

		return errors.New(cc.status.statusString())
	}

	return nil
}

// getStatus - get the current status of the connection
func (cc *Client) getStatus() Status {

	return cc.status

}

// Status - returns the current connection status as a string
func (cc *Client) Status() string {

	return cc.status.statusString()

}

// Close - closes the connection
func (cc *Client) Close() {
	cc.status = Closing
	cc.conn.Close()
}
