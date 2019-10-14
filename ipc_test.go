package ipc

import (
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

	scon.Timeout = -1
	scon.MaxMsgSize = -1

	_, err5 := StartServer("test", scon)
	if err5 != nil {
		t.Error(err2)
	}

	ccon.Timeout = -1
	ccon.RetryTimer = -1
	ccon.MaxMsgSize = 0

	_, err6 := StartClient("test", ccon)
	if err6 != nil {
		t.Error(err)
	}

	scon.MaxMsgSize = 1025
	ccon.MaxMsgSize = 1025
	ccon.RetryTimer = 1

	_, err7 := StartServer("test", scon)
	if err7 != nil {
		t.Error(err2)
	}

	_, err8 := StartClient("test", ccon)
	if err8 != nil {
		t.Error(err)
	}
}

func TestStartUp_Timeout(t *testing.T) {

	scon := &ServerConfig{
		Timeout: 1,
	}

	sc, _ := StartServer("test_dummy", scon)

	_, _, err := sc.Read()

	if err.Error() != "Timed out waiting for client to connect" {
		t.Error("should of got server timeout")
	}

	ccon := &ClientConfig{
		Timeout:    2,
		RetryTimer: 1,
	}

	cc, _ := StartClient("test2", ccon)

	_, _, err2 := cc.Read()

	if err2.Error() != "Timed out trying to connect" {
		t.Error("should of got timeout as client was trying to connect")
	}

}

func TestWrite(t *testing.T) {

	// 3 x Server side tests
	sc := &Server{
		name:       "Test_write",
		status:     Connected,
		recieved:   make(chan Message),
		timeout:    0,
		maxMsgSize: maxMsgSize,
	}

	buf := make([]byte, 1)

	err := sc.Write(0, buf)
	if err.Error() != "Message type 0 is reserved" {
		t.Error("0 is not allowed as a message type")
	}

	buf = make([]byte, sc.maxMsgSize+5)
	err2 := sc.Write(2, buf)

	if err2.Error() != "Message exceeds maximum message length" {
		t.Error("There should be an error as the data we're attempting to write is bigger than the maxMsgSize")
	}

	sc.status = NotConnected

	buf2 := make([]byte, 5)
	err3 := sc.Write(2, buf2)
	if err3.Error() == "Not Connected" {

	} else {
		t.Error("we should have an error becuse there is no connection")
	}

	// 3 x client side tests
	cc := &Client{
		Name:       "test_Write",
		timeout:    0,
		retryTimer: 1,
		status:     Connected,
		maxMsgSize: maxMsgSize,
	}

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
		recieved: make(chan Message),
		timeout:  0,
	}

	sIPC.status = Connected

	testMessage := Message{msgType: 1}

	go func(s *Server) {

		_, _, err4 := sIPC.Read()
		if err4 != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to recieved")
		}
		_, _, err5 := sIPC.Read()
		if err5 != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to recieved")
		}

		_, _, err6 := sIPC.Read()
		if err6 == nil {
			t.Error("we should get an error as the messages have been read and the channel closed")
		}

	}(sIPC)

	sIPC.recieved <- testMessage // add 1st message
	sIPC.recieved <- testMessage // add 2nd message
	close(sIPC.recieved)         // close channel

	// Client - read tests

	// 3 x client side tests
	cIPC := &Client{
		Name:       "test",
		timeout:    0,
		retryTimer: 1,
		status:     NotConnected,
		recieved:   make(chan Message),
	}

	cIPC.status = Connected

	testMessage2 := Message{msgType: 1}

	go func() {

		_, _, err11 := cIPC.Read()
		if err11 != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to recieved")
		}
		_, _, err12 := cIPC.Read()
		if err12 != nil {
			t.Error("err should be nill as tbe read function should read the 1st message added to recieved")
		}

		_, _, err13 := cIPC.Read()
		if err13 == nil {
			t.Error("we should get an error as the messages have been read and the channel closed")
		}

	}()

	cIPC.recieved <- testMessage2 // add 1st message
	cIPC.recieved <- testMessage2 // add 2nd message
	close(cIPC.recieved)          // close channel

}

func TestStatus(t *testing.T) {

	sc := &Server{
		status: NotConnected,
	}

	s := sc.getStatus()

	if s.statusString() != "Not Connected" {
		t.Error("status string should have returned Not Connected")
	}

	sc.status = Listening

	s1 := sc.getStatus()

	if s1.statusString() != "Listening" {
		t.Error("status string should have returned Listening")
	}

	sc.status = Connected

	s2 := sc.getStatus()

	if s2.statusString() != "Connected" {
		t.Error("status string should have returned Connected")
	}

	sc.status = ReConnecting

	s3 := sc.getStatus()

	if s3.statusString() != "Re-connecting" {
		t.Error("status string should have returned Re-connecting")
	}

	sc.status = Closed

	s4 := sc.getStatus()

	if s4.statusString() != "Closed" {
		t.Error("status string should have returned Closed")
	}

	sc.status = Error

	s5 := sc.getStatus()

	if s5.statusString() != "Error" {
		t.Error("status string should have returned Error")
	}

	sc.Status()

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

func checkStatus(sc *Server, t *testing.T) bool {

	for i := 0; i < 25; i++ {

		if sc.getStatus() == 3 {
			return true

		} else if i == 25 {
			t.Error("Server failed to connect")
			break
		}

		time.Sleep(time.Second / 5)
	}

	return false

}

func TestGetConnected(t *testing.T) {

	sc, err := StartServer("test22", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	_, err2 := StartClient("test22", nil)
	if err2 != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i++ {

		if sc.getStatus() == Connected {

			break

		} else if i == 9 {
			t.Error("Server failed to connect")
			break
		}

		time.Sleep(time.Second)
	}

}

func TestServerReadWrite(t *testing.T) {

	sc, err := StartServer("test3", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test3", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	// test wrong message type
	cc.Write(2, []byte("hello server 1"))
	msgType, _, err := sc.Read()

	if err == nil {
		if msgType != 5 {
			// recieved wrong message type
		} else {
			t.Error("should have got wrong message type")
		}
	} else {
		t.Error(err)
	}

	// test correct message type
	cc.Write(5, []byte("hello server 5"))
	msgType, _, err3 := sc.Read()

	if err3 == nil {

		if msgType == 5 {
			// good message type is 5
		} else {
			t.Error(" wrong message type recieved")
		}

	} else {
		t.Error(err3)
	}

	// test recieving a message
	cc.Write(5, []byte("hello server 3 -/and some more test data to pad it out a bit"))
	msgType, msg, err3 := sc.Read()

	if err3 == nil {

		if msgType == 5 {
			// good message type is 5
			if string(msg) == "hello server 3 -/and some more test data to pad it out a bit" {
				// correct msg has been recieved
			} else {
				t.Error("Message recieved is wrong")
			}

		} else {
			t.Error(" wrong message type recieved")
		}

	} else {
		t.Error(err)
	}

}

func TestClientReadWrite(t *testing.T) {

	sc, err := StartServer("test4", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test4", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	// test wrong message type
	sc.Write(2, []byte("hello client"))
	msgType, _, err := cc.Read()

	if err == nil {
		if msgType != 5 {
			// recieved wrong message type
		} else {
			t.Error("should have got wrong message type")
		}
	} else {
		t.Error(err)
	}

	// test correct message type
	sc.Write(5, []byte("hello client"))
	msgType, _, err3 := cc.Read()

	if err3 == nil {

		if msgType == 5 {
			// good message type is 5
		} else {
			t.Error(" wrong message type recieved")
		}

	} else {
		t.Error(err3)
	}

	// test recieving a message
	sc.Write(5, []byte("hello client -/and some more test data to pad it out a bit"))
	msgType, msg, err3 := cc.Read()

	if err3 == nil {

		if msgType == 5 {
			// good message type is 5
			if string(msg) == "hello client -/and some more test data to pad it out a bit" {
				// correct msg has been recieved
			} else {
				t.Error("Message recieved is wrong")
			}

		} else {
			t.Error(" wrong message type recieved")
		}

	} else {
		t.Error(err)
	}

}

func TestClientWrongVersionNumber(t *testing.T) {

	sc, err := StartServer("test5", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test5", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	message := []byte("data")

	cHeader := createHeader(2, 3, false, 0)

	write := append(cHeader, message...)

	cc.conn.Write(write)

	_, _, err3 := sc.Read()
	if err3 == nil {

		t.Error("Should have client sent wrong version number")

	}

	_, _, err4 := cc.Read()
	if err4 == nil {
		t.Error("Should have server sent wrong version number")
	}

}

func TestServerWrongVersionNumber(t *testing.T) {

	sc, err := StartServer("test6", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test6", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	message := []byte("data")

	cHeader := createHeader(2, 3, false, 0)

	write := append(cHeader, message...)

	sc.conn.Write(write)

	_, _, err3 := cc.Read()
	if err3 == nil {
		t.Error("Should have server sent wrong version number")
	}

	_, _, err4 := sc.Read()
	if err4 == nil {
		t.Error("Should have server sent wrong version number")
	}

}

func TestReconnect(t *testing.T) {

	sc, err := StartServer("test7", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test7", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	sc.status = Closing
	sc.listen.Close()
	sc.conn.Close()

	time.Sleep(time.Second / 2)
	if cc.status != ReConnecting {
		t.Error("client should be trying to re-connect")
	}

	sc, err4 := StartServer("test7", nil)

	if err4 != nil {
		t.Error(err4)
	}

	time.Sleep(time.Second * 3)

	if cc.status != Connected {
		t.Error("client should be connected")
	}

	cc.conn.Close()
	time.Sleep(time.Second / 4)

	if sc.getStatus() != ReConnecting {
		t.Error("server should be trying to re-connect")
	}

	cc, err5 := StartClient("test7", nil)

	if err5 != nil {
		t.Error(err5)
	}

	time.Sleep(time.Second)

	if sc.getStatus() != Connected {
		t.Error("server should be connected")
	}

}

func TestTimeoutReconnect(t *testing.T) {

	//server closes, client times out

	scon := &ServerConfig{
		Timeout: 1,
	}

	ccon := &ClientConfig{
		Timeout:    2,
		RetryTimer: 1,
	}

	sc, err := StartServer("test_timeout", scon)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test_timeout", ccon)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	sc.Close()

	_, _, err5 := cc.Read()
	if err5.Error() != "Timed out trying to re-connect" {
		t.Error("client should have timed out")
	}

	//client closes, server times out

	scon2 := &ServerConfig{
		Timeout: 2,
	}

	ccon2 := &ClientConfig{
		Timeout:    2,
		RetryTimer: 1,
	}

	sc2, err8 := StartServer("test_timeout2", scon2)
	if err8 != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc2, err9 := StartClient("test_timeout2", ccon2)
	if err9 != nil {
		t.Error(err)
	}

	if checkStatus(sc2, t) == false {
		t.FailNow()
	}

	cc2.Close()

	_, _, err10 := sc2.Read()
	if err10.Error() != "Timed out trying to re-connect" {
		t.Error("server should have timed out")
	}

}

func TestMultiMessageType(t *testing.T) {

	sc, err := StartServer("test8", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test8", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	message := []byte("data")

	cHeader := createHeader(1, 5, true, 123456789)

	write := append(cHeader, message...)

	sc.conn.Write(write)

	_, _, err3 := cc.Read()
	if err3 != nil {
		t.Error("error with server sending multi message")
	}

	cc.conn.Write(write)

	_, _, err4 := sc.Read()
	if err4 != nil {
		t.Error("error with client sending multi message")
	}

}

func TestServerClose(t *testing.T) {

	sc, err := StartServer("test9", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	_, err2 := StartClient("test9", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	sc.Close()

	_, _, err3 := sc.Read()
	if err3.Error() != "Connection closed" {
		t.Error(err3)
	}

}

func TestClientClose(t *testing.T) {

	sc, err := StartServer("test10", nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test10", nil)
	if err2 != nil {
		t.Error(err)
	}

	if checkStatus(sc, t) == false {
		t.FailNow()
	}

	cc.Close()

	_, _, err4 := cc.Read()
	if err4.Error() != "Connection closed" {
		t.Error(err4)
	}

}
