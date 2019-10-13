package ipc

import (
	"testing"
	"time"
)

func TestStartUp_Name(t *testing.T) {

	_, err := StartServer("", -1)
	if err == nil {
		t.Error(err)
	}

	_, err2 := StartClient("", -1, -1)
	if err2 == nil {
		t.Error(err)
	}

}

func TestStartUp_Timers(t *testing.T) {

	_, err := StartServer("test", -1)
	if err != nil {
		t.Error(err)
	}

	_, err2 := StartServer("test", -1)
	if err2 != nil {
		t.Error(err2)
	}

	_, err3 := StartClient("test", -1, -1)
	if err3 != nil {
		t.Error(err)
	}

	_, err4 := StartClient("test", -1, 0)
	if err4 != nil {
		t.Error(err)
	}

}

func TestStartUp_Timeout(t *testing.T) {

	sc, _ := StartServer("test", 1)

	_, _, err := sc.Read()

	if err == nil {
		t.Error(err)
	}

	cc, _ := StartClient("test2", 1, 1)

	_, _, err2 := cc.Read()

	if err2 == nil {
		t.Error(err2)
	}

}

func TestWrite(t *testing.T) {

	// 3 x Server side tests
	sIPC := &Server{
		name:     "Test",
		status:   NotConnected,
		recieved: make(chan Message),
		timeout:  0,
	}

	buf := make([]byte, 1)

	err := sIPC.Write(0, buf)
	if err == nil {
		t.Error("0 is not allowwed as a message type")
	}

	err = sIPC.Write(1, buf)
	if err == nil {
		t.Error("1 is not allowwed as a message type")
	}

	buf = make([]byte, maxMsgSize+5)
	err = sIPC.Write(2, buf)
	if err == nil {
		t.Error("There should be an error as the data we're attempting to write is bigger than the maxMsgSize")
	}

	buf = make([]byte, 5)
	err = sIPC.Write(2, buf)
	if err.Error() == "Not Connected" {

	} else {
		t.Error("we should have an error becuse there is no connection")
	}

	// 3 x client side tests
	cIPC := &Client{
		Name:       "test",
		timeout:    0,
		retryTimer: 1,
		status:     NotConnected,
		//	Recieved:   make(chan Message),
	}

	buf = make([]byte, 1)

	err = cIPC.Write(0, buf)
	if err == nil {
		t.Error("0 is not allowwed as a message try")
	}

	err = cIPC.Write(1, buf)
	if err == nil {
		t.Error("0 is not allowwed as a message try")
	}

	buf = make([]byte, maxMsgSize+5)
	err = cIPC.Write(2, buf)
	if err == nil {
		t.Error("There should be an error is the data we're attempting to write is bigger than the maxMsgSize")
	}

	buf = make([]byte, 5)
	err = cIPC.Write(2, buf)
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

}
func connect(t *testing.T) (sc *Server, cc *Client, err error) {

	sc, err = StartServer("test", 4)
	if err != nil {
		return nil, nil, err
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test", 4, 1)
	if err2 != nil {
		return nil, nil, err2
	}

	return sc, cc, err

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

	sc, err := StartServer("test22", 2)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	_, err2 := StartClient("test22", 2, 1)
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

	sc, err := StartServer("test3", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test3", 4, 1)
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

	cc.conn.Close()
	_, _, err4 := sc.Read()

	if err4 == nil {
		t.Error("Read should return an error as the client has closed the connection")
	} else {
		if err4.Error() != "Timed out trying to re-connect" {
			t.Error("should have timed out trying to re-connect")
		}
	}
}

func TestClientReadWrite(t *testing.T) {

	sc, err := StartServer("test4", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test4", 4, 1)
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

	sc.status = Closing
	sc.listen.Close()
	sc.conn.Close()

	_, _, err4 := cc.Read()

	if err4 == nil {
		t.Error("Read should return an error as the client has closed the connection")
	} else {
		if err4.Error() != "Timed out try to re-connect" {
			t.Error("should have timed out trying to re-connect")
		}
	}

}

func TestClientWrongVersionNumber(t *testing.T) {

	sc, err := StartServer("test5", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test5", 4, 1)
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

	sc, err := StartServer("test6", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test6", 4, 1)
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

	sc, err := StartServer("test7", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test7", 4, 1)
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

	sc, err4 := StartServer("test7", 2)

	if err4 != nil {
		t.Error(err4)
	}

	time.Sleep(time.Second)

	if cc.status != Connected {
		t.Error("client should be connected")
	}

	cc.conn.Close()
	time.Sleep(time.Second / 4)

	if sc.getStatus() != ReConnecting {
		t.Error("server should be trying to re-connect")
	}

	cc, err5 := StartClient("test7", 2, 1)

	if err5 != nil {
		t.Error(err5)
	}

	time.Sleep(time.Second)

	if sc.getStatus() != Connected {
		t.Error("server should be connected")
	}

}

func TestMultiMessageType(t *testing.T) {

	sc, err := StartServer("test8", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test8", 4, 1)
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

	sc, err := StartServer("test9", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	_, err2 := StartClient("test9", 4, 1)
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

	sc, err := StartServer("test10", 4)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second / 4)

	cc, err2 := StartClient("test10", 4, 1)
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
