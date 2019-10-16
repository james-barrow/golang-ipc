package ipc

import (
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
		recieved: make(chan Message),
	}

	if config == nil {
		sc.timeout = 0
		sc.maxMsgSize = maxMsgSize

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
	}

	go startServer(sc)

	return sc, err
}

func startServer(sc *Server) {

	err := sc.run()
	if err != nil {
		sc.recieved <- Message{err: err, msgType: 0}
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
			go sc.read()
			sc.status = Connected
			sc.connChannel <- true
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

	header := make([]byte, 9)
	buff := make([]byte, sc.maxMsgSize)

	for {

		result := sc.readData(header)
		if result == false {
			break
		}

		ph := processHeader(header)

		if ph.msgType == 0 {

			result := sc.readData(buff[:ph.msgLen])
			if result == false {
				break
			}

			sc.msgTypeZero(buff[:ph.msgLen])

		} else {

			if ph.version != version {
				sc.writeControlMessage("WV")
				sc.status = Error
				sc.listen.Close()
				sc.conn.Close()
				sc.recieved <- Message{err: errors.New("Client sent the wrong version number"), msgType: 0}
				break
			}

			result := sc.readData(buff[:ph.msgLen])
			if result == false {
				break
			}

			sc.recieved <- Message{data: buff[:ph.msgLen], msgType: ph.msgType}

		}

	}
}

func (sc *Server) readData(buff []byte) bool {

	_, err := sc.conn.Read(buff)
	if err != nil {

		if sc.status == Closing {
			sc.recieved <- Message{err: errors.New("Connection closed"), msgType: 0}
			sc.status = Closed
			close(sc.recieved)
			return false
		}

		sc.status = ReConnecting
		go sc.reConnect()
		return false

	}

	return true

}

func (sc *Server) msgTypeZero(buf []byte) {

	// Wrong version error.
	if string(buf) == "WV" {
		sc.recieved <- Message{err: errors.New("Client recieved wrong version number"), msgType: 0}
	}

}

func (sc *Server) writeControlMessage(mess string) {

	message := []byte(mess)

	header := createHeader(version, 0, len(message))
	header = append(header, message...)

	_, _ = sc.conn.Write(header)
}

func (sc *Server) reConnect() {

	err := sc.connectionTimer()
	if err != nil {
		sc.status = Error
		sc.recieved <- Message{err: errors.New("Timed out trying to re-connect"), msgType: 0}
		close(sc.recieved)

	}
}

// Read - blocking function that waits until an non multipart message is recieved

func (sc *Server) Read() (uint32, []byte, error) {

	m, ok := (<-sc.recieved)
	if ok == false {
		return 0, nil, errors.New("the recieve channel has been closed")
	}

	return m.msgType, m.data, m.err

}

// Write - writes a non multipart message to the ipc connection.
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
//
func (sc *Server) Write(msgType uint32, message []byte) error {

	if msgType == 0 {
		return errors.New("Message type 0 is reserved")
	}

	mlen := len(message)

	if mlen > sc.maxMsgSize {
		return errors.New("Message exceeds maximum message length")
	}

	if sc.status == Connected {

		header := createHeader(version, msgType, mlen)
		header = append(header, message...)

		_, err := sc.conn.Write(header)

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
