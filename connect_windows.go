package ipc

import (
	"errors"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
)

// Server function
// Create the named pipe (if it doesn't already exist) and start listening for a client to connect.
// when a client connects and connection is accepted the read function is called on a go routine.
func (sc *Server) run() error {

	var pipeBase = `\\.\pipe\`

	listen, err := winio.ListenPipe(pipeBase+sc.name, nil)
	if err != nil {

		return err
	}

	sc.listen = listen

	sc.status = Listening

	sc.connChannel = make(chan bool)

	go sc.acceptLoop()

	err2 := sc.connectionTimer()
	if err2 != nil {
		return err2
	}

	return nil

}

// Client function
// dial - attempts to connect to a named pipe created by the server
func (cc *Client) dial() error {

	var pipeBase = `\\.\pipe\`

	startTime := time.Now()

	for {
		if cc.timeout != 0 {
			if time.Now().Sub(startTime).Seconds() > cc.timeout {
				cc.status = Closed
				return errors.New("Timed out trying to connect")
			}
		}
		pn, err := winio.DialPipe(pipeBase+cc.Name, nil)
		if err != nil {

			if strings.Contains(err.Error(), "The system cannot find the file specified.") == true {

			} else {
				return err
			}

		} else {

			cc.conn = pn

			err = cc.handshake()
			if err != nil {
				return err
			}
			return nil
		}

		time.Sleep(cc.retryTimer * time.Second)

	}
}
