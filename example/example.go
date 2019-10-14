package main

import (
	"log"
	"time"

	ipc "../../golang-ipc"
)

func main() {

	go server()

	client()

}

func server() {

	ipcm, err := ipc.StartServer("testtest", nil)
	if err != nil {
		log.Println(err)
		return
	}

	go readServerRecv(ipcm)

	for {

		_ = ipcm.Write(5, []byte("hello client"))

		time.Sleep(1 * time.Second)

	}
}

func readServerRecv(s *ipc.Server) {

	for {
		mt, data, err := s.Read()

		if err == nil {
			log.Println("Server recieved: "+string(data)+" - Message type: ", mt)
		} else {

			log.Println("Server error")
			log.Println(err)
			break
		}
	}
}

func client() {

	ipcc, err := ipc.StartClient("testtest", nil)
	if err != nil {
		log.Println(err)
		return
	}

	go readClientRecv(ipcc)

	for {

		_ = ipcc.Write(1, []byte("hello server"))

		time.Sleep(time.Second / 2)

	}

}

func readClientRecv(c *ipc.Client) {

	for {
		mt, data, err := c.Read()

		if err != nil {
			log.Println("Client error")
			log.Println(err)
			break
		} else {
			log.Println("Client recieved: "+string(data)+" - Message type: ", mt)
		}
	}

}
