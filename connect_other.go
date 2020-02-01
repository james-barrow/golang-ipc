// +build linux darwin

package ipc

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"
)

// Server create a unix socket and start listening connections - for unix and linux
func (sc *Server) run() error {

	base := "/tmp/"
	sock := ".sock"

	if err := os.RemoveAll(base + sc.name + sock); err != nil {
		return err
	}

	listen, err := net.Listen("unix", base+sc.name+sock)
	if err != nil {
		return err
	}

	sc.listen = listen

	sc.status = Listening
	sc.recieved <- &Message{Status: sc.status.String(), MsgType: -1}
	sc.connChannel = make(chan bool)

	go sc.acceptLoop()

	err = sc.connectionTimer()
	if err != nil {
		return err
	}

	return nil

}

// Client connect to the unix socket created by the server -  for unix and linux
func (cc *Client) dial() error {

	base := "/tmp/"
	sock := ".sock"

	startTime := time.Now()

	for {
		if cc.timeout != 0 {
			if time.Now().Sub(startTime).Seconds() > cc.timeout {
				cc.status = Closed
				return errors.New("Timed out trying to connect")
			}
		}

		conn, err := net.Dial("unix", base+cc.Name+sock)
		if err != nil {

			if strings.Contains(err.Error(), "connect: no such file or directory") == true {

			} else if strings.Contains(err.Error(), "connect: connection refused") == true {

			} else {
				cc.recieved <- &Message{err: err, MsgType: -2}
			}

		} else {

			cc.conn = conn

			err = cc.handshake()
			if err != nil {
				return err
			}

			return nil
		}

		time.Sleep(cc.retryTimer * time.Second)

	}

}
