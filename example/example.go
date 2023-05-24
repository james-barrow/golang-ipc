package main

import (
	"log"

	ipc "github.com/james-barrow/golang-ipc"
)

func main() {

	go server()

	c, err := ipc.StartClient("example1", nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {

		message, err := c.Read()

		if err == nil {

			if message.MsgType == -1 {

				log.Println("client status", c.Status())

				if message.Status == "Reconnecting" {
					c.Close()
					return
				}

			} else {

				log.Println("Client received: "+string(message.Data)+" - Message type: ", message.MsgType)
				c.Write(5, []byte("Message from client - PONG"))

			}

		} else {
			log.Println(err)
			break
		}
	}

}

func server() {

	s, err := ipc.StartServer("example1", nil)
	if err != nil {
		log.Println("server error", err)
		return
	}

	log.Println("server status", s.Status())

	for {

		message, err := s.Read()

		if err == nil {

			if message.MsgType == -1 {

				if message.Status == "Connected" {

					log.Println("server status", s.Status())
					s.Write(1, []byte("server - PING"))

				}

			} else {

				log.Println("Server received: "+string(message.Data)+" - Message type: ", message.MsgType)
				s.Close()
				return
			}

		} else {
			break
		}
	}

}
