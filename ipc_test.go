package ipc

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func TestStartUp_Name(t *testing.T) {

	_, err := StartServer("", nil)
	if err.Error() != "ipcName cannot be an empty string" {
		t.Error("server - should have an error becuse the ipc name is empty")
	}
	_, err2 := StartClient("", nil)
	if err2.Error() != "ipcName cannot be an empty string" {
		t.Error("client - should have an error becuse the ipc name is empty")
	}
}

func TestStartUp_Configs(t *testing.T) {

	_, err := StartServer("test", nil)
	if err != nil {
		t.Error(err)
	}

	_, err2 := StartClient("test", nil)
	if err2 != nil {
		t.Error(err)
	}

	scon := &ServerConfig{}

	ccon := &ClientConfig{}

	_, err3 := StartServer("test", scon)
	if err3 != nil {
		t.Error(err2)
	}

	_, err4 := StartClient("test", ccon)
	if err4 != nil {
		t.Error(err)
	}

	scon.MaxMsgSize = -1

	_, err5 := StartServer("test", scon)
	if err5 != nil {
		t.Error(err2)
	}

	ccon.Timeout = -1
	ccon.RetryTimer = -1

	_, err6 := StartClient("test", ccon)
	if err6 != nil {
		t.Error(err)
	}

	scon.MaxMsgSize = 1025
	ccon.RetryTimer = 1

	_, err7 := StartServer("test", scon)
	if err7 != nil {
		t.Error(err2)
	}

	_, err8 := StartClient("test", ccon)
	if err8 != nil {
		t.Error(err)
	}

	t.Run("Unmask Server Socket Permissions", func(t *testing.T) {
		scon.UnmaskPermissions = true

		srv, err := StartServer("test_perm", scon)
		if err != nil {
			t.Error(err)
		}

		// test would not work in windows
		// can check test_perm.sock in /tmp after running tests to see perms

		time.Sleep(time.Second / 4)

		info, err := os.Stat(srv.listen.Addr().String())
		if err != nil {
			t.Error(err)
		}
		got := fmt.Sprintf("%04o", info.Mode().Perm())
		want := "0777"

		if got != want {
			t.Errorf("Got %q, Wanted %q", got, want)
		}

		scon.UnmaskPermissions = false
	})
}

/*
func TestStartUp_Timeout(t *testing.T) {

	scon := &ServerConfig{
		Timeout: 1,
	}

	sc, _ := StartServer("test_dummy", scon)

	for {
		_, err1 := sc.Read()

		if err1 != nil {
			if err1.Error() != "timed out waiting for client to connect" {
				t.Error("should of got server timeout")
			}
			break
		}

	}

	ccon := &ClientConfig{
		Timeout:    2,
		RetryTimer: 1,
	}

	cc, _ := StartClient("test2", ccon)

	for {
		_, err := cc.Read()
		if err != nil {
			if err.Error() != "timed out trying to connect" {
				t.Error("should of got timeout as client was trying to connect")
			}

			break

		}
	}

}
*/

func TestWrite(t *testing.T) {

	sc, err := StartServer("test10", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test10", nil)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)

	go func() {

		for {

			m, _ := cc.Read()
			if m.Status == "Connected" {
				connected <- true
			}
		}
	}()

	go func() {

		for {
			sc.Read()
		}

	}()

	<-connected

	buf := make([]byte, 1)

	err3 := sc.Write(0, buf)
	if err3.Error() != "message type 0 is reserved" {
		t.Error("0 is not allowed as a message type")
	}

	buf = make([]byte, sc.maxMsgSize+5)
	err4 := sc.Write(2, buf)

	if err4.Error() != "message exceeds maximum message length" {
		t.Error("There should be an error as the data we're attempting to write is bigger than the maxMsgSize")
	}

	sc.status = NotConnected

	buf2 := make([]byte, 5)
	err5 := sc.Write(2, buf2)
	if err5.Error() != "Not Connected" {
		t.Error("we should have an error becuse there is no connection")
	}

	sc.status = Connected

	buf = make([]byte, 1)

	err = cc.Write(0, buf)
	if err == nil {
		t.Error("0 is not allowwed as a message try")
	}

	buf = make([]byte, maxMsgSize+5)
	err = cc.Write(2, buf)
	if err == nil {
		t.Error("There should be an error is the data we're attempting to write is bigger than the maxMsgSize")
	}

	cc.status = NotConnected

	buf = make([]byte, 5)
	err = cc.Write(2, buf)
	if err.Error() == "Not Connected" {

	} else {
		t.Error("we should have an error becuse there is no connection")
	}
}

func TestRead(t *testing.T) {

	sIPC := &Server{
		name:     "Test",
		status:   NotConnected,
		received: make(chan *Message),
		timeout:  0,
	}

	sIPC.status = Connected

	serverFinished := make(chan bool, 1)

	go func(s *Server) {

		_, err := sIPC.Read()
		if err != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to received")
		}
		_, err2 := sIPC.Read()
		if err2 != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to received")
		}

		_, err3 := sIPC.Read()
		if err3 == nil {
			t.Error("we should get an error as the messages have been read and the channel closed")

		} else {
			serverFinished <- true
		}

	}(sIPC)

	sIPC.received <- &Message{MsgType: 1, Data: []byte("message 1")}
	sIPC.received <- &Message{MsgType: 1, Data: []byte("message 2")}
	close(sIPC.received) // close channel

	<-serverFinished

	// Client - read tests

	// 3 x client side tests
	cIPC := &Client{
		Name:       "test",
		timeout:    2,
		retryTimer: 1,
		status:     NotConnected,
		received:   make(chan *Message),
	}

	cIPC.status = Connected

	clientFinished := make(chan bool, 1)

	go func() {

		_, err4 := cIPC.Read()
		if err4 != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to received")
		}
		_, err5 := cIPC.Read()
		if err5 != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to received")
		}

		_, err6 := cIPC.Read()
		if err6 == nil {
			t.Error("we should get an error as the messages have been read and the channel closed")
		} else {
			clientFinished <- true
		}

	}()

	cIPC.received <- &Message{MsgType: 1, Data: []byte("message 1")}
	cIPC.received <- &Message{MsgType: 1, Data: []byte("message 1")}
	close(cIPC.received) // close received channel

	<-clientFinished
}

func TestStatus(t *testing.T) {

	sc := &Server{
		status: NotConnected,
	}

	s := sc.getStatus()

	if s.String() != "Not Connected" {
		t.Error("status string should have returned Not Connected")
	}

	sc.status = Listening

	s1 := sc.getStatus()

	if s1.String() != "Listening" {
		t.Error("status string should have returned Listening")
	}

	sc.status = Connecting

	s1 = sc.getStatus()

	if s1.String() != "Connecting" {
		t.Error("status string should have returned Connecting")
	}

	sc.status = Connected

	s2 := sc.getStatus()

	if s2.String() != "Connected" {
		t.Error("status string should have returned Connected")
	}

	sc.status = ReConnecting

	s3 := sc.getStatus()

	if s3.String() != "Reconnecting" {
		t.Error("status string should have returned Reconnecting")
	}

	sc.status = Closed

	s4 := sc.getStatus()

	if s4.String() != "Closed" {
		t.Error("status string should have returned Closed")
	}

	sc.status = Error

	s5 := sc.getStatus()

	if s5.String() != "Error" {
		t.Error("status string should have returned Error")
	}

	sc.status = Closing

	s6 := sc.getStatus()

	if s6.String() != "Closing" {
		t.Error("status string should have returned Error")
	}

	if s6.String() != "Closing" {
		t.Error("status string should have returned Error")
	}

	sc.status = 33

	s7 := sc.getStatus()

	fmt.Println(s7.String())
	if s7.String() != "Status not found" {
		t.Error("status string should have returned 'Status not found'")
	}

	cc := &Client{
		status: NotConnected,
	}

	cc.getStatus()
	cc.Status()

	cc2 := &Client{
		status: 9,
	}

	cc2.getStatus()
	cc2.Status()
}

func TestGetConnected(t *testing.T) {

	sc, err := StartServer("test22", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 2)

	cc, err2 := StartClient("test22", nil)
	if err2 != nil {
		t.Error(err)
	}

	for {
		cc.Read()
		m, _ := sc.Read()

		if m.Status == "Connected" {
			break
		}
	}
}

func TestServerWrongMessageType(t *testing.T) {

	sc, err := StartServer("test333", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test333", nil)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)
	connected2 := make(chan bool, 1)
	complete := make(chan bool, 1)

	go func() {

		ready := false

		for {
			m, _ := sc.Read()
			if m.Status == "Connected" {
				connected <- true
				ready = true
				continue
			}

			if ready == true {
				if m.MsgType != 5 {
					// received wrong message type

				} else {
					t.Error("should have got wrong message type")
				}
				complete <- true
				break
			}
		}

	}()

	go func() {
		for {
			m, _ := cc.Read()

			if m.Status == "Connected" {
				connected2 <- true
			}
		}
	}()

	<-connected
	<-connected2

	// test wrong message type
	cc.Write(2, []byte("hello server 1"))

	<-complete
}
func TestClientWrongMessageType(t *testing.T) {

	sc, err := StartServer("test3", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test3", nil)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)
	connected2 := make(chan bool, 1)
	complete := make(chan bool, 1)

	go func() {
		for {
			m, _ := sc.Read()
			if m.Status == "Connected" {
				connected2 <- true
				continue

			}

		}
	}()

	go func() {

		ready := false

		for {

			m, err45 := cc.Read()

			if m.Status == "Connected" {
				connected <- true
				ready = true
				continue

			}

			if ready == true {

				if err45 == nil {
					if m.MsgType != 5 {
						// received wrong message type
					} else {
						t.Error("should have got wrong message type")
					}
					complete <- true
					break

				} else {
					t.Error(err45)
					break
				}
			}

		}
	}()

	<-connected
	<-connected2
	sc.Write(2, []byte(""))

	<-complete
}
func TestServerCorrectMessageType(t *testing.T) {

	sc, err := StartServer("test358", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test358", nil)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)
	connected2 := make(chan bool, 1)
	complete := make(chan bool, 1)

	go func() {
		for {
			m, _ := sc.Read()
			if m.Status == "Connected" {
				connected2 <- true
			}
		}
	}()

	go func() {

		ready := false

		for {

			m, err23 := cc.Read()

			if m.Status == "Connected" {
				ready = true
				connected <- true
				continue
			}

			if ready == true {
				if err23 == nil {
					if m.MsgType == 5 {
						// received correct message type
					} else {
						t.Error("should have got correct message type")
					}

					complete <- true

				} else {
					t.Error(err23)
					break
				}
			}

		}
	}()

	<-connected
	<-connected2

	sc.Write(5, []byte(""))

	<-complete
}

func TestClientCorrectMessageType(t *testing.T) {

	sc, err := StartServer("test355", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test355", nil)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)
	connected2 := make(chan bool, 1)
	complete := make(chan bool, 1)

	go func() {

		for {
			m, _ := cc.Read()

			if m.Status == "Connected" {
				connected2 <- true
			}
		}

	}()

	go func() {

		ready := false

		for {

			m, err34 := sc.Read()

			if m.Status == "Connected" {
				ready = true
				connected <- true
				continue
			}

			if ready == true {
				if err34 == nil {
					if m.MsgType == 5 {
						// received correct message type
					} else {
						t.Error("should have got correct message type")
					}

					complete <- true

				} else {
					t.Error(err34)
					break
				}
			}
		}
	}()

	<-connected2
	<-connected

	cc.Write(5, []byte(""))
	<-complete
}
func TestServerSendMessage(t *testing.T) {

	sc, err := StartServer("test377", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test377", nil)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)
	connected2 := make(chan bool, 1)
	complete := make(chan bool, 1)

	go func() {

		for {

			m, _ := sc.Read()

			if m.Status == "Connected" {
				connected <- true
			}
		}
	}()

	go func() {

		ready := false

		for {

			m, err56 := cc.Read()

			if m.Status == "Connected" {
				ready = true
				connected2 <- true
				continue
			}

			if ready == true {
				if err56 == nil {
					if m.MsgType == 5 {
						if string(m.Data) == "Here is a test message sent from the server to the client... -/and some more test data to pad it out a bit" {
							// correct msg has been received
						} else {
							t.Error("Message recreceivedieved is wrong")
						}
					} else {
						t.Error("should have got correct message type")
					}

					complete <- true
					break

				} else {
					t.Error(err56)
					complete <- true
					break
				}

			}
		}

	}()

	<-connected2
	<-connected

	sc.Write(5, []byte("Here is a test message sent from the server to the client... -/and some more test data to pad it out a bit"))

	<-complete
}
func TestClientSendMessage(t *testing.T) {

	sc, err := StartServer("test3661", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test3661", nil)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)
	connected2 := make(chan bool, 1)
	complete := make(chan bool, 1)

	go func() {

		for {

			m, _ := cc.Read()
			if m.Status == "Connected" {
				connected <- true
			}

		}
	}()

	go func() {

		ready := false

		for {

			m, _ := sc.Read()

			if m.Status == "Connected" {
				ready = true
				connected2 <- true
				continue
			}

			if ready == true {
				if err == nil {
					if m.MsgType == 5 {

						if string(m.Data) == "Here is a test message sent from the client to the server... -/and some more test data to pad it out a bit" {
							// correct msg has been received
						} else {
							t.Error("Message recreceivedieved is wrong")
						}

					} else {
						t.Error("should have got correct message type")
					}
					complete <- true
					break

				} else {
					t.Error(err)
					complete <- true
					break
				}
			}

		}
	}()

	<-connected
	<-connected2

	cc.Write(5, []byte("Here is a test message sent from the client to the server... -/and some more test data to pad it out a bit"))

	<-complete
}

func TestEncryptionFunctions(t *testing.T) {

	res := publicKeyToBytes(nil)
	if len(res) != 0 {
		t.Error("should have returned 0 bytes")

	}

	buff := make([]byte, 0)

	if bytesToPublicKey(buff) != nil {
		t.Error("should have failed as buff is 0 bytes")
	}



}

func TestNoEncrytion(t *testing.T) {

	config := &ServerConfig{Encryption: false}

	sc, err := StartServer("test11", config)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	config2 := &ClientConfig{Encryption: false}

	cc, err2 := StartClient("test11", config2)
	if err2 != nil {
		t.Error(err)
	}

	connected := make(chan bool, 1)
	connected2 := make(chan bool, 1)
	complete := make(chan bool, 1)
	complete2 := make(chan bool, 1)

	go func() {
		for {
			m, err := sc.Read()

			if m.Status == "Connected" {
				connected <- true
				continue
			}

			if err != nil {
				t.Error(err)
			} else if string(m.Data) == "Message to server" {
				complete2 <- true
			}

		}
	}()

	go func() {
		for {

			m, err := cc.Read()

			if m.Status == "Connected" {
				connected2 <- true
				continue
			}

			if err != nil {
				t.Error(err)
			} else if string(m.Data) == "Message to client" {
				complete <- true
			}
		}
	}()

	<-connected
	<-connected2

	sc.Write(2, []byte("Message to client"))
	cc.Write(2, []byte("Message to server"))

	<-complete
	<-complete2
}
func TestServerWrongEncrytion(t *testing.T) {

	config := &ServerConfig{Encryption: false}

	sc, err := StartServer("test11", config)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	config2 := &ClientConfig{Encryption: true}

	cc, err2 := StartClient("test11", config2)
	if err2 != nil {
		t.Error(err)
	}

	go func() {
		for {
			m, err := cc.Read()
			if err != nil {
				if err.Error() != "server tried to connect without encryption" && m.MsgType != -2 {
					t.Error(err)
				}
				break
			}
		}
	}()

	for {
		mm, err2 := sc.Read()
		if err2 != nil {
			if err2.Error() != "client is enforcing encryption" && mm.MsgType != -2 {
				t.Error(err2)
			} else {
				break
			}
			break
		}
	}
}

func TestClientClose(t *testing.T) {

	sc, err := StartServer("test10A", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test10A", nil)
	if err2 != nil {
		t.Error(err)
	}

	holdIt := make(chan bool, 1)

	go func() {

		for {

			m, _ := sc.Read()

			if m.Status == "Disconnected" {
				holdIt <- false
				break
			}

		}

	}()

	for {

		mm, err := cc.Read()

		if err == nil {
			if mm.Status == "Connected" {
				cc.Close()
			}

			if mm.Status == "Closed" {
				break
			}
		}

	}

	<-holdIt
}

func TestServerClose(t *testing.T) {

	sc, err := StartServer("test1010", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test1010", nil)
	if err2 != nil {
		t.Error(err)
	}

	holdIt := make(chan bool, 1)

	go func() {

		for {

			m, _ := cc.Read()

			if m.Status == "Reconnecting" {
				holdIt <- false
				break
			}
		}

	}()

	for {

		mm, err2 := sc.Read()

		if err2 == nil {
			if mm.Status == "Connected" {
				sc.Close()
			}

			if mm.Status == "Closed" {
				break
			}
		}

	}

	<-holdIt
}

func TestClientReconnect(t *testing.T) {

	sc, err := StartServer("test127", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test127", nil)
	if err2 != nil {
		t.Error(err)
	}
	connected := make(chan bool, 1)
	clientConfirm := make(chan bool, 1)
	clientConnected := make(chan bool, 1)

	go func() {

		for {

			m, _ := sc.Read()
			if m.Status == "Connected" {
				connected <- true
				break
			}

		}
	}()

	go func() {

		reconnectCheck := 0

		for {

			m, _ := cc.Read()
			if m.Status == "Connected" {
				clientConnected <- true
			}

			if m.Status == "Reconnecting" {
				reconnectCheck = 2
			}

			if m.Status == "Connected" && reconnectCheck == 2 {
				clientConfirm <- true
				break
			}
		}
	}()

	<-connected
	<-clientConnected

	sc.Close()

	sc2, err := StartServer("test127", nil)
	if err != nil {
		t.Error(err)
	}

	for {

		m, _ := sc2.Read()
		if m.Status == "Connected" {
			<-clientConfirm
			break
		}
	}
}

func TestClientReconnectTimeout(t *testing.T) {

	server, err := StartServer("test7", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	config := &ClientConfig{
		Timeout:    2,
		RetryTimer: 1,
	}

	cc, err2 := StartClient("test7", config)
	if err2 != nil {
		t.Error(err)
	}

	go func() {

		for {

			m, _ := server.Read()
			if m.Status == "Connected" {
				server.Close()
				break
			}

		}
	}()

	connect := false
	reconnect := false

	for {

		mm, err5 := cc.Read()

		if err5 == nil {
			if mm.Status == "Connected" {
				connect = true
			}

			if mm.Status == "Reconnecting" {
				reconnect = true
			}

			if mm.Status == "Timeout" && reconnect == true && connect == true {
				return
			}
		}

		if err5 != nil {
			if err5.Error() != "timed out trying to re-connect" {
				t.Fatal("should have got the timed out error")
			}

			break

		}
	}
}

func TestServerReconnect(t *testing.T) {

	sc, err := StartServer("test127", nil)
	if err != nil {
		t.Error(err)
	}

	cc, err2 := StartClient("test127", nil)
	if err2 != nil {
		t.Error(err2)
	}
	connected := make(chan bool, 1)
	clientConfirm := make(chan bool, 1)
	clientConnected := make(chan bool, 1)

	go func() {

		for {

			m, _ := cc.Read()
			if m.Status == "Connected" {
				<-clientConnected

				connected <- true
				break
			}

		}
	}()

	go func() {

		reconnectCheck := 0

		for {

			m, err := sc.Read()
			if err != nil {
				fmt.Println(err)
				return
			}

			if m.Status == "Connected" {
				clientConnected <- true
			}

			if m.Status == "Disconnected" {
				reconnectCheck = 1
			}

			if m.Status == "Connected" && reconnectCheck == 1 {
				clientConfirm <- true
				break
			}
		}
	}()

	<-connected

	cc.Close()

	c2, err := StartClient("test127", nil)
	if err != nil {
		t.Error(err)
	}

	for {

		m, _ := c2.Read()
		if m.Status == "Connected" {
			break
		}
	}

	<-clientConfirm
}

func TestServerReconnect2(t *testing.T) {

	sc, err := StartServer("test337", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test337", nil)
	if err2 != nil {
		t.Error(err)
	}

	hasConnected := make(chan bool)
	hasDisconnected := make(chan bool)
	hasReconnected := make(chan bool)

	go func() {

		for {

			m, _ := cc.Read()
			if m.Status == "Connected" {

				<-hasConnected

				cc.Close()

				<-hasDisconnected

				c2, err2 := StartClient("test337", nil)
				if err2 != nil {
					t.Error(err)
				}

				for {

					m, _ := c2.Read()
					if m.Status == "Connected" {
						break
					}
				}

				<-hasReconnected

				return
			}

		}
	}()

	connect := false
	disconnect := false

	for {

		m, _ := sc.Read()
		if m.Status == "Connected" && connect == false {
			hasConnected <- true
			connect = true

		}

		if m.Status == "Disconnected" {
			hasDisconnected <- true
			disconnect = true

		}

		if m.Status == "Connected" && connect == true && disconnect == true {
			hasReconnected <- true
			return
		}
	}
}

func TestClientReadClose(t *testing.T) {

	sc, err := StartServer("test7R", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	config := &ClientConfig{
		Timeout:    2,
		RetryTimer: 1,
	}

	cc, err2 := StartClient("test7R", config)
	if err2 != nil {
		t.Error(err)
	}
	connected := make(chan bool, 1)
	clientTimout := make(chan bool, 1)
	clientConnected := make(chan bool, 1)
	clientError := make(chan bool, 1)

	go func() {

		for {

			m, _ := sc.Read()
			if m.Status == "Connected" {
				connected <- true
				break
			}

		}
	}()

	go func() {

		reconnect := false

		for {

			m, err3 := cc.Read()

			if err3 != nil {
				if err3.Error() == "the received channel has been closed" {
					clientError <- true // after the connection times out the received channel is closed, so we're now testing that the close error is returned.
					// This is the only error the received function returns.
					break
				}
			}

			if err3 == nil {
				if m.Status == "Connected" {
					clientConnected <- true
				}

				if m.Status == "Reconnecting" {
					reconnect = true
				}

				if m.Status == "Timeout" && reconnect == true {
					clientTimout <- true
				}
			}

		}
	}()

	<-connected
	<-clientConnected

	sc.Close()

	<-clientTimout
	<-clientError
}

func TestServerReceiveWrongVersionNumber(t *testing.T) {

	sc, err := StartServer("test5", nil)
	if err != nil {
		t.Error(err)
	}

	go func() {

		cc := &Client{
			Name:          "",
			status:        NotConnected,
			received:      make(chan *Message),
			encryptionReq: false,
		}

		time.Sleep(3 * time.Second)

		base := "/tmp/"
		sock := ".sock"
		conn, _ := net.Dial("unix", base+"test5"+sock)

		cc.conn = conn

		recv := make([]byte, 2)
		_, err2 := cc.conn.Read(recv)
		if err2 != nil {
			return
		}

		if recv[0] != 4 {
			cc.handshakeSendReply(1)
			return
		}

	}()

	for {

		m, err := sc.Read()
		if err != nil {
			fmt.Println(err)
			return
		}

		if m.Err != nil {
			if m.Err.Error() != "client has a different version number" {
				t.Error("should have error because server sent the client the wrong version number 1")
			}
		}
	}
}

func TestClientReceiveWrongVersionNumber(t *testing.T) {

	//conn, err := s.listen.Accept()
	//if err != nil {
	//	break
	//}
}

/*
// This test will not pass on Windows unless the net.Listen part for unix sockets is replaced with winio.ListenPipe
func TestServerWrongVersionNumber(t *testing.T) {

	sc := &Server{
		name:     "test6",
		status:   NotConnected,
		received: make(chan *Message),
	}

	go func() {

		//var pipeBase = `\\.\pipe\`

		//listen, err := winio.ListenPipe(pipeBase+sc.name, nil)
		//if err != nil {

		//	return err
		//}

		//sc.listen = listen

		base := "/tmp/"
		sock := ".sock"

		os.RemoveAll(base + sc.name + sock)

		sc.listen, _ = net.Listen("unix", base+sc.name+sock)

		conn, _ := sc.listen.Accept()
		sc.conn = conn

		buff := make([]byte, 2)

		buff[0] = byte(8)

		buff[1] = byte(0)

		_, err := sc.conn.Write(buff)
		if err != nil {
			//return errors.New("unable to send handshake ")
		}

		m := sc.Read()
		if m.Err == nil {
			t.Error("Should have server sent wrong version number")
		}

	}()

	time.Sleep(time.Second)

	cc, err := StartClient("test6", nil)
	if err != nil {
		t.Error(err)
	}

	mm := cc.Read()
	if mm.Err.Error() != "server has sent a different version number" {
		t.Error("Should have  wrong version number")
	}

}

*/
