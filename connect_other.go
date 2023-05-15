//go:build linux || darwin
// +build linux darwin

package ipc

import (
	"errors"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

// Server create a unix socket and start listening connections - for unix and linux
func (s *Server) run() error {

	base := "/tmp/"
	sock := ".sock"

	if err := os.RemoveAll(base + s.name + sock); err != nil {
		return err
	}

	var oldUmask int
	if s.unMask {
		oldUmask = syscall.Umask(0)
	}

	listen, err := net.Listen("unix", base+s.name+sock)

	if s.unMask {
		syscall.Umask(oldUmask)
	}

	if err != nil {
		return err
	}

	s.listen = listen

	s.status = Listening
	s.received <- &Message{Status: s.status.String(), MsgType: -1}
	s.connChannel = make(chan bool)

	go s.acceptLoop()

	err = s.connectionTimer()
	if err != nil {
		return err
	}

	return nil

}

// Client connect to the unix socket created by the server -  for unix and linux
func (c *Client) dial() error {

	base := "/tmp/"
	sock := ".sock"

	startTime := time.Now()

	for {
		if c.timeout != 0 {

			if time.Now().Sub(startTime).Seconds() > c.timeout {
				c.status = Closed
				return errors.New("timed out trying to connect")
			}
		}

		conn, err := net.Dial("unix", base+c.Name+sock)
		if err != nil {

			if strings.Contains(err.Error(), "connect: no such file or directory") {

			} else if strings.Contains(err.Error(), "connect: connection refused") {

			} else {
				c.received <- &Message{err: err, MsgType: -2}
			}

		} else {

			c.conn = conn

			err = c.handshake()
			if err != nil {
				return err
			}

			return nil
		}

		time.Sleep(c.retryTimer * time.Second)

	}

}
