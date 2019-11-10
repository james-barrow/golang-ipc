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
		sc.recieved <- &Message{err: err, msgType: 0}
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
				sc.recieved <- &Message{err: err2, msgType: 0}
				sc.status = Error
				sc.listen.Close()
				sc.conn.Close()

			} else {
				go sc.read()
				sc.status = Connected
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
				break
			}

			sc.recieved <- &Message{data: msgFinal[4:], msgType: bytesToInt(msgFinal[:4])}

		} else {
			sc.recieved <- &Message{data: msgRecvd[4:], msgType: bytesToInt(msgRecvd[:4])}
		}

	}
}

func (sc *Server) readData(buff []byte) bool {

	_, err := sc.conn.Read(buff)
	if err != nil {

		if sc.status == Closing {
			sc.recieved <- &Message{err: errors.New("Connection closed"), msgType: 0}
			sc.status = Closed
			close(sc.recieved)
			return false
		}

		sc.status = ReConnecting
		go sc.reConnect()
		return false

	}

	//if sc.encryption == true {
	//	retbuff, err := decrypt(sc.gcm, buff)
	//	if err != nil {
	//		log.Println(err)
	//	}

	//	buff = retbuff

	//}

	return true

}

func (sc *Server) reConnect() {

	err := sc.connectionTimer()
	if err != nil {
		sc.status = Error
		sc.recieved <- &Message{err: errors.New("Timed out trying to re-connect"), msgType: 0}
		close(sc.recieved)

	}
}

// Read - blocking function that waits until an non multipart message is recieved

func (sc *Server) Read() (int, []byte, error) {

	m, ok := (<-sc.recieved)
	if ok == false {
		return 0, nil, errors.New("the recieve channel has been closed")
	}

	return m.msgType, m.data, m.err

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

		toSend := intToBytes(msgType)

		writer := bufio.NewWriter(sc.conn)

		if sc.encryption == true {
			toSend = append(toSend, message...)
			toSendEnc, err := encrypt(*sc.enc.cipher, toSend)
			if err != nil {
				return err
			}
			toSend = toSendEnc
		} else {

			toSend = append(toSend, message...)

		}

		writer.Write(intToBytes(len(toSend)))
		writer.Write(toSend)

		err := writer.Flush()
		if err != nil {
			return err
		}

	} else {
		return errors.New(sc.status.statusString())
	}

	return nil

}

// getStatus - get the current status of the connection
func (sc *Server) getStatus() Status {

	return sc.status

}

// Status - returns the current connection status as a string
func (sc *Server) Status() string {

	return sc.status.statusString()

}

// Close - closes the connection
func (sc *Server) Close() {

	sc.status = Closing
	sc.listen.Close()
	sc.conn.Close()

}
