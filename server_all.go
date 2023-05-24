package ipc

import (
	"bufio"
	"errors"
	"io"
	"log"
	"time"
)

// StartServer - starts the ipc server.
//
// ipcName - is the name of the unix socket or named pipe that will be created, the client needs to use the same name
func StartServer(ipcName string, config *ServerConfig) (*Server, error) {

	err := checkIpcName(ipcName)
	if err != nil {
		return nil, err
	}

	s := &Server{
		name:     ipcName,
		status:   NotConnected,
		received: make(chan *Message),
		toWrite:  make(chan *Message),
	}

	if config == nil {
		s.timeout = 0
		s.maxMsgSize = maxMsgSize
		s.encryption = true
		s.unMask = false

	} else {

		if config.MaxMsgSize < 1024 {
			s.maxMsgSize = maxMsgSize
		} else {
			s.maxMsgSize = config.MaxMsgSize
		}

		if !config.Encryption {
			s.encryption = false
		} else {
			s.encryption = true
		}

		if config.UnmaskPermissions {
			s.unMask = true
		} else {
			s.unMask = false
		}
	}

	err = s.run()

	return s, err
}

func (s *Server) acceptLoop() {

	for {
		conn, err := s.listen.Accept()
		if err != nil {
			break
		}

		if s.status == Listening || s.status == Disconnected {

			s.conn = conn

			err2 := s.handshake()
			if err2 != nil {
				s.received <- &Message{Err: err2, MsgType: -1}
				s.status = Error
				s.listen.Close()
				s.conn.Close()

			} else {

				go s.read()
				go s.write()

				s.status = Connected
				s.received <- &Message{Status: s.status.String(), MsgType: -1}
			}

		}

	}

}

func (s *Server) read() {

	bLen := make([]byte, 4)

	for {

		res := s.readData(bLen)
		if !res {
			s.conn.Close()

			break
		}

		mLen := bytesToInt(bLen)

		msgRecvd := make([]byte, mLen)

		res = s.readData(msgRecvd)
		if !res {
			s.conn.Close()

			break
		}

		if s.encryption {
			msgFinal, err := decrypt(*s.enc.cipher, msgRecvd)
			if err != nil {
				s.received <- &Message{Err: err, MsgType: -1}
				continue
			}

			if bytesToInt(msgFinal[:4]) == 0 {
				//  type 0 = control message
			} else {
				s.received <- &Message{Data: msgFinal[4:], MsgType: bytesToInt(msgFinal[:4])}
			}

		} else {
			if bytesToInt(msgRecvd[:4]) == 0 {
				//  type 0 = control message
			} else {
				s.received <- &Message{Data: msgRecvd[4:], MsgType: bytesToInt(msgRecvd[:4])}
			}
		}

	}

}

func (s *Server) readData(buff []byte) bool {

	_, err := io.ReadFull(s.conn, buff)
	if err != nil {

		if s.status == Closing {

			s.status = Closed
			s.received <- &Message{Status: s.status.String(), MsgType: -1}
			s.received <- &Message{Err: errors.New("server has closed the connection"), MsgType: -1}
			return false
		}

		if err == io.EOF {

			s.status = Disconnected
			s.received <- &Message{Status: s.status.String(), MsgType: -1}
			return false
		}

	}

	return true
}

// Read - blocking function, reads each message recieved
// if MsgType is a negative number its an internal message
func (s *Server) Read() (*Message, error) {

	m, ok := (<-s.received)
	if !ok {
		return nil, errors.New("the received channel has been closed")
	}

	if m.Err != nil {
		//close(s.received)
		//close(s.toWrite)
		return nil, m.Err
	}

	return m, nil
}

// Write - writes a message to the ipc connection
// msgType - denotes the type of data being sent. 0 is a reserved type for internal messages and errors.
func (s *Server) Write(msgType int, message []byte) error {

	if msgType == 0 {
		return errors.New("message type 0 is reserved")
	}

	mlen := len(message)

	if mlen > s.maxMsgSize {
		return errors.New("message exceeds maximum message length")
	}

	if s.status == Connected {

		s.toWrite <- &Message{MsgType: msgType, Data: message}

	} else {
		return errors.New(s.status.String())
	}

	return nil
}

func (s *Server) write() {

	for {

		m, ok := <-s.toWrite

		if !ok {
			break
		}

		toSend := intToBytes(m.MsgType)

		writer := bufio.NewWriter(s.conn)

		if s.encryption {
			toSend = append(toSend, m.Data...)
			toSendEnc, err := encrypt(*s.enc.cipher, toSend)
			if err != nil {
				log.Println("error encrypting data", err)
				continue
			}

			toSend = toSendEnc
		} else {

			toSend = append(toSend, m.Data...)

		}

		writer.Write(intToBytes(len(toSend)))
		writer.Write(toSend)

		err := writer.Flush()
		if err != nil {
			log.Println("error flushing data", err)
			continue
		}

		time.Sleep(2 * time.Millisecond)

	}
}


// getStatus - get the current status of the connection
func (s *Server) getStatus() Status {

	return s.status
}


// StatusCode - returns the current connection status
func (s *Server) StatusCode() Status {
	return s.status
}

// Status - returns the current connection status as a string
func (s *Server) Status() string {

	return s.status.String()
}

// Close - closes the connection
func (s *Server) Close() {

	s.status = Closing

	if s.listen != nil {
		s.listen.Close()
	}

	if s.conn != nil {
		s.conn.Close()
	}
}
