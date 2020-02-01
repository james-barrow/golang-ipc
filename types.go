package ipc

import (
	"crypto/cipher"
	"net"
	"time"
)

// Server - holds the details of the server connection & config.
type Server struct {
	name        string
	listen      net.Listener
	conn        net.Conn
	status      Status
	recieved    chan (*Message)
	connChannel chan bool
	toWrite     chan (*Message)
	timeout     time.Duration
	encryption  bool
	maxMsgSize  int
	enc         *encryption
}

// Client - holds the details of the client connection and config.
type Client struct {
	Name          string
	conn          net.Conn
	status        Status
	timeout       float64       //
	retryTimer    time.Duration // number of seconds before trying to connect again
	recieved      chan (*Message)
	toWrite       chan (*Message)
	encryption    bool
	encryptionReq bool
	maxMsgSize    int
	enc           *encryption
}

// Message - contains the  recieved message
type Message struct {
	err     error  // details of any error
	MsgType int    // type of message sent - 0 is reserved
	Data    []byte // message data recieved
	Status  string
}

// Status - Status of the connection
type Status int

const (

	// NotConnected - 0
	NotConnected Status = iota
	// Listening - 1
	Listening Status = iota
	// Connecting - 2
	Connecting Status = iota
	// Connected - 3
	Connected Status = iota
	// ReConnecting - 4
	ReConnecting Status = iota
	// Closed - 5
	Closed Status = iota
	// Closing - 6
	Closing Status = iota
	// Error - 7
	Error Status = iota
	// Timeout - 8
	Timeout Status = iota
)

// ServerConfig - used to pass configuation overrides to ServerStart()
type ServerConfig struct {
	Timeout    time.Duration
	MaxMsgSize int
	Encryption bool
}

// ClientConfig - used to pass configuation overrides to ClientStart()
type ClientConfig struct {
	Timeout    float64
	RetryTimer time.Duration
	Encryption bool
}

// Encryption - encryption settings
type encryption struct {
	keyExchange string
	encryption  string
	cipher      *cipher.AEAD
}
