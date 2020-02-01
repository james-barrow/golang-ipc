package main

import (
	"log"
	"time"

	ipc "../../golang-ipc"
)

func main() {

	log.Println("starting")

	server()

	//client()

}

func server() {

	//&ipc.ServerConfig{Encryption: false}

	sc, err := ipc.StartServer("testtest", nil)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {

		for {
			m, err := sc.Read()

			if err == nil {
				if m.MsgType > 0 {
					log.Println("Server recieved: "+string(m.Data)+" - Message type: ", m.MsgType)
				}

			} else {

				log.Println("Server error")
				log.Println(err)
				break
			}
		}
	}()

	go serverSend(sc)
	go serverSend1(sc)
	serverSend2(sc)

}

func serverSend(sc *ipc.Server) {

	for {

		err := sc.Write(3, []byte("Hello Client 4"))
		err = sc.Write(23, []byte("Hello Client 5"))
		err = sc.Write(65, []byte("Hello Client 6"))

		if err != nil {
			//fmt.Println(err)
		}

		time.Sleep(time.Second / 30)

	}
}

func serverSend1(sc *ipc.Server) {

	for {

		sc.Write(5, []byte("Hello Client 1"))
		sc.Write(7, []byte("Hello Client 2"))
		sc.Write(9, []byte("Hello Client 3"))

		time.Sleep(time.Second / 30)

	}

}

func serverSend2(sc *ipc.Server) {

	for {

		err := sc.Write(88, []byte("Hello Client 7"))
		err = sc.Write(99, []byte("Hello Client 8"))
		err = sc.Write(22, []byte("Hello Client 9"))

		if err != nil {
			//fmt.Println(err)
		}

		time.Sleep(time.Second / 30)

	}
}

func client() {

	//config := &ipc.ClientConfig{Encryption: false}

	cc, err := ipc.StartClient("testtest", nil)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {

		for {
			m, err := cc.Read()

			if err != nil {
				// An error is only returned if the recieved channel has been closed,
				//so you know the connection has either been intentionally closed or has timmed out waiting to connect/re-connect.
				break
			}

			if m.MsgType == -1 { // message type -1 is status change
				//log.Println("Status: " + m.Status)
			}

			if m.MsgType == -2 { // message type -2 is an error, these won't automatically cause the recieve channel to close.
				log.Println("Error: " + err.Error())
			}

			if m.MsgType > 0 { // all message types above 0 have been recieved over the connection

				log.Println(" Message type: ", m.MsgType)
				log.Println("Client recieved: " + string(m.Data))
			}
			//}
		}

	}()

	go clientSend(cc)
	go clientSend(cc)
	clientSend2(cc)

}

func clientSend(cc *ipc.Client) {

	for {

		_ = cc.Write(14, []byte("hello server 4"))
		_ = cc.Write(44, []byte("hello server 5"))
		_ = cc.Write(88, []byte("hello server 6"))

		time.Sleep(time.Second / 20)

	}

}

func clientSend1(cc *ipc.Client) {

	for {

		_ = cc.Write(1, []byte("hello server 1"))
		_ = cc.Write(9, []byte("hello server 2"))
		_ = cc.Write(34, []byte("hello server 3"))

		time.Sleep(time.Second / 20)

	}

}

func clientSend2(cc *ipc.Client) {

	for {

		_ = cc.Write(444, []byte("hello server 7"))
		_ = cc.Write(234, []byte("hello server 8"))
		_ = cc.Write(111, []byte("hello server 9"))

		time.Sleep(time.Second / 20)

	}
}

func clientRecv(c *ipc.Client) {

	for {
		m, err := c.Read()

		if err != nil {
			// An error is only returned if the recieved channel has been closed,
			//so you know the connection has either been intentionally closed or has timmed out waiting to connect/re-connect.
			break
		}

		if m.MsgType == -1 { // message type -1 is status change
			//log.Println("Status: " + m.Status)
		}

		if m.MsgType == -2 { // message type -2 is an error, these won't automatically cause the recieve channel to close.
			log.Println("Error: " + err.Error())
		}

		if m.MsgType > 0 { // all message types above 0 have been recieved over the connection

			log.Println(" Message type: ", m.MsgType)
			log.Println("Client recieved: " + string(m.Data))
		}
		//}
	}

}
