# golang-ipc
 Golang Inter-process communication library for Window, Mac and Linux.


 ### Overview
 
 A simple to use package that uses unix sockets on Macos/Linux and named pipes on Windows to create a communication channel between two go processes.

### Intergration

As well as using this library just for go processes it was also designed to work with other languages, with the go process as the server and the other languages processing being the client.


#### NodeJs

I currently use this library to comunicate between a ElectronJS GUI and a go program.

Below is a link to the nodeJS client library:

https://github.com/james-barrow/node-ipc-client

#### Python 

To do

## Usage

Create a server with the default configuation and start listening for the client:

```go

	s, err := ipc.StartServer("<name of socket or pipe>", nil)
	if err != nil {
		log.Println(err)
		return
	}

```
Create a client and connect to the server:

```go

	c, err := ipc.StartClient("<name of socket or pipe>", nil)
	if err != nil {
		log.Println(err)
		return
	}

```

### Read messages 

Read each message sent:

```go

    for {

        // message, err := s.Read() server 
        message, err := c.Read() // client

        if err == nil {
            // handle error
        }

        // do something with the received messages
    }

```

All received messages are formated into the type Message

```go

type Message struct {
	Err     error  // details of any error
	MsgType int    // 0 = reserved , -1 is an internal message (disconnection or error etc), all messages recieved will be > 0
	Data    []byte // message data received
	Status  string // the status of the connection
}

```

### Write a message


```go

	//err := s.Write(1, []byte("<Message for client"))
    err := c.Write(1, []byte("<Message for server"))

    if err == nil {
        // handle error
    }

```

 ## Advanced Configuaration

Server options:

```go

    config := &ipc.ServerConfig{
		Encryption: (bool),        // allows encryption to be switched off (bool - default is true)
        MaxMsgSize: (int) ,        // the maximum size in bytes of each message ( default is 3145728 / 3Mb)
	    UnmaskPermissions: (bool), // make the socket writeable for other users (default is false)
    }


```

Client options:

```go

	config := ClientConfig  {
		Encryption (bool),          // allows encryption to be switched off (bool - default is true)
		Timeout    (float64),       // number of seconds to wait before timing out trying to connect/reconnect (default is 0 no timeout)
		RetryTimer (time.Duration), // number of seconds to wait before connection retry (default is 20)
		
	}

```

 ### Encryption

 By default the connection established will be encypted, ECDH384 is used for the key exchange and AES 256 GCM is used for the cipher.

 Encryption can be swithed off by passing in a custom configuation to the server & client start function:

```go
	Encryption: false
```

 ### Unix Socket Permissions

 Under most configurations, a socket created by a user will by default not be writable by another user, making it impossible for the client and server to communicate if being run by separate users.

 The permission mask can be dropped during socket creation by passing a custom configuration to the server start function.  **This will make the socket writable for any user.**

```go
	UnmaskPermissions: true	
```
 Note: Tested on Linux, not tested on Mac, not implemented on Windows.
 


 ## Testing

 The package has been tested on Mac, Windows and Linux and has extensive test coverage.

## Licence

MIT
