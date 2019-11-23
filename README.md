# golang-ipc
 Golang Inter-process communication library for Window, Mac and Linux.


 ### Overview
 
 A simple to use package that uses unix sockets on Macos/Linux and named pipes on Windows to create a communication channel between two go processes.


## Usage

Create a server with the default configuation and start listening for the client:

```go

	sc, err := ipc.StartServer("<name of socket or pipe>", nil)
	if err != nil {
		log.Println(err)
		return
	}

```

Create a client and connect to the server:

```go

	cc, err := ipc.StartClient("<name of socket or pipe>", nil)
	if err != nil {
		log.Println(err)
		return
	}

```
Read and write data to the connection:

```go
        // write data
        _ = sc.Write(1, []byte("Message from server"))
        
        _ = cc.Write(5, []byte("Message from client"))


        // Read data
        for {
            
            dataType, data, err := sc.Read()

            if err == nil {
                log.Println("Server recieved: "+string(data)+" - Message type: ", dataType)
            } else {
                log.Println(err)
                break
            }
	    }


        for {
            
            dataType, data, err := cc.Read()

            if err == nil {
                log.Println("Client recieved: "+string(data)+" - Message type: ", dataType)     
            } else {
                log.Println(err)
                break
            }
	    }

```

 ### Encryption

 By default the connection established will be encypted, ECDH384 is used for the key exchange and AES 256 GCM is used for the cipher.

 Encryption can be swithed off by passing in a custom configuation to the server & client start functions.

```go
    
    config := &ipc.ServerConfig{Encryption: false}
	sc, err := ipc.StartServer("<name of socket or pipe>", config)

```

 ### Testing

 The package has been tested on Mac, Windows and Linux and has extensive test coverage.

### Licence

MIT