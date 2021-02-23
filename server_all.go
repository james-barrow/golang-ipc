package ipc

import (
	"bufio"
	"errors"
	"time"
)

// StartServer - starts the ipc server.
//
// ipcName = is the name of the unix socket or named pipe that will be created.
// timeout = number of seconds before the socket/pipe times out waiting for a connection/re-cconnection - if -1 or 0 it never times out.
//
func StartServer(ipcName string, config *ServerConfig) (*Server, error) {

	err := checkIpcName(ipcName)
	if err != nil {
		return nil, err
	}

	sc := &Server{
		name:     ipcName,
		status:   NotConnected,
		recieved: make(chan *Message),
		toWrite:  make(chan *Message),
	}

	if config == nil {
		sc.timeout = 0
		sc.maxMsgSize = maxMsgSize
		sc.encryption = true

	} else {

		if config.Timeout < 0 {
			sc.timeout = 0
		} else {
			sc.timeout = config.Timeout
		}

		if config.MaxMsgSize < 1024 {
			sc.maxMsgSize = maxMsgSize
		} else {
			sc.maxMsgSize = config.MaxMsgSize
		}

		if config.Encryption == false {
			sc.encryption = false
		} else {
			sc.encryption = true
		}

	}

	go startServer(sc)

	return sc, err
}

func startServer(sc *Server) {

	err := sc.run()
	if err != nil {
		sc.recieved <- &Message{err: err, MsgType: -2}
	}
}

func (sc *Server) acceptLoop() {
	for {
		conn, err := sc.listen.Accept()
		if err != nil {
			break
		}

		if sc.status == Listening || sc.status == ReConnecting {

			sc.conn = conn

			err2 := sc.handshake()
			if err2 != nil {
				sc.recieved <- &Message{err: err2, MsgType: -2}
				sc.status = Error
				sc.listen.Close()
				sc.conn.Close()

			} else {
				go sc.read()
				go sc.write()

				sc.status = Connected
				sc.recieved <- &Message{Status: sc.status.String(), MsgType: -1}
				sc.connChannel <- true
			}

		}

	}

}

func (sc *Server) connectionTimer() error {

	if sc.timeout != 0 {

		timeout := make(chan bool)

		go func() {
			time.Sleep(sc.timeout * time.Second)
			timeout <- true
		}()

		select {

		case <-sc.connChannel:
			return nil
		case <-timeout:
			sc.listen.Close()
			return errors.New("Timed out waiting for client to connect")
		}
	}

	select {

	case <-sc.connChannel:
		return nil
	}

}

func (sc *Server) read() {

	bLen := make([]byte, 4)

	for {

		res := sc.readData(bLen)
		if res == false {
			break
		}

		mLen := bytesToInt(bLen)

		msgRecvd := make([]byte, mLen)

		res = sc.readData(msgRecvd)
		if res == false {
			break
		}

		if sc.encryption == true {
			msgFinal, err := decrypt(*sc.enc.cipher, msgRecvd)
			if err != nil {
				sc.recieved <- &Message{err: err, MsgType: -2}
				continue
			}

			if bytesToInt(msgFinal[:4]) == 0 {
				//  type 0 = control message
			} else {
				sc.recieved <- &Message{Data: msgFinal[4:], MsgType: bytesToInt(msgFinal[:4])}
			}

		} else {
			if bytesToInt(msgRecvd[:4]) == 0 {
				//  type 0 = control message
			} else {
				sc.recieved <- &Message{Data: msgRecvd[4:], MsgType: bytesToInt(msgRecvd[:4])}
			}
		}

	}
}

func (sc *Server) readData(buff []byte) bool {

	_, err := sc.conn.Read(buff)
	if err != nil {

		if sc.status == Closing {

			sc.status = Closed
			sc.recieved <- &Message{Status: sc.status.String(), MsgType: -1}
			sc.recieved <- &Message{err: errors.New("Server has closed the connection"), MsgType: -2}
			return false
		}

		go sc.reConnect()
		return false

	}

	return true

}

func (sc *Server) reConnect() {

	sc.status = ReConnecting
	sc.recieved <- &Message{Status: sc.status.String(), MsgType: -1}

	err := sc.connectionTimer()
	if err != nil {
		sc.status = Timeout
		sc.recieved <- &Message{Status: sc.status.String(), MsgType: -1}

		sc.recieved <- &Message{err: err, MsgType: -2}

	}

}

// Read - blocking function that waits until an non multipart message is recieved

func (sc *Server) Read() (*Message, error) {

	m, ok := (<-sc.recieved)
	if ok == false {
		return nil, errors.New("the recieve channel has been closed")
	}

	if m.err != nil {
		close(sc.recieved)
		close(sc.toWrite)
		return nil, m.err
	}

	return m, nil

}

// Write - writes a non multipart message to the ipc connection.
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
//
func (sc *Server) Write(msgType int, message []byte) error {

	if msgType == 0 {
		return errors.New("Message type 0 is reserved")
	}

	mlen := len(message)

	if mlen > sc.maxMsgSize {
		return errors.New("Message exceeds maximum message length")
	}

	if sc.status == Connected {

		sc.toWrite <- &Message{MsgType: msgType, Data: message}

	} else {
		return errors.New(sc.status.String())
	}

	return nil

}

func (sc *Server) write() {

	for {

		m, ok := <-sc.toWrite

		if ok == false {
			break
		}

		toSend := intToBytes(m.MsgType)

		writer := bufio.NewWriter(sc.conn)

		if sc.encryption == true {
			toSend = append(toSend, m.Data...)
			toSendEnc, err := encrypt(*sc.enc.cipher, toSend)
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

		time.Sleep(2 * time.Millisecond)

	}

}

// getStatus - get the current status of the connection
func (sc *Server) getStatus() Status {

	return sc.status

}

// StatusCode - returns the current connection status
func (sc *Server) StatusCode() Status {
	return sc.status
}

// Status - returns the current connection status as a string
func (sc *Server) Status() string {

	return sc.status.String()

}

// Close - closes the connection
func (sc *Server) Close() {

	sc.status = Closing
	sc.listen.Close()
	sc.conn.Close()

}
