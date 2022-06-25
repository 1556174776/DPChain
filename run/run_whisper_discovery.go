// Contains a simple whisper peer setup and self messaging to allow playing
// around with the protocol and API without a fancy client implementation.
// using the discovery

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"p2p-go/common"
	"p2p-go/crypto"
	"p2p-go/discover"
	"p2p-go/logger"
	"p2p-go/nat"
	"p2p-go/server"
	"p2p-go/whisper"
)

func main() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))
	// logger.NewLogger("TAG").Infoln("Start Whisper Server...")

	shh1 := whisper.New()
	shh2 := whisper.New()
	shh3 := whisper.New()

	server1 := createServer("serv1", nil, shh1.Protocol(), "127.0.0.1:30300")
	url1 := server1.Self().String()
	n, err := discover.ParseNode(url1)
	if err != nil {
		fmt.Printf("Error: Bootstrap URL %s: %v\n", url1, err)
		os.Exit(-1)
	}
	server2 := createServer("serv2", []*discover.Node{n}, shh2.Protocol(), "127.0.0.1:30301")
	server3 := createServer("serv3", []*discover.Node{n}, shh3.Protocol(), "127.0.0.1:30302")
	url2 := server2.Self().String()
	url3 := server3.Self().String()

	fmt.Println("The URL of server1:", url1)
	fmt.Println("The URL of server2:", url2)
	fmt.Println("The URL of server3:", url3)

	// Send a message to self to check that something works
	payload1 := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", server2.Name, time.Now())
	if err := selfSend(shh1, shh2, []byte(payload1)); err != nil {
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}
	payload2 := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", server1.Name, time.Now())
	if err := selfSend(shh2, shh1, []byte(payload2)); err != nil {
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}

	payload3 := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", server3.Name, time.Now())
	if err := selfSend(shh2, shh3, []byte(payload3)); err != nil {
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}

	fmt.Println(server1.ReadRandomNodes(5))
	fmt.Println(server2.ReadRandomNodes(5))
	fmt.Println(server3.ReadRandomNodes(5))
	server1.Stop()
	server2.Stop()
	server3.Stop()
}

func createServer(servername string, boostNodes []*discover.Node, pro server.Protocol, addr string) *server.Server {
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Printf("Failed to generate peer key: %v.\n", err)
		os.Exit(-1)
	}
	name := common.MakeName("whisper-go-"+servername, "1.0")

	server := server.Server{
		PrivateKey:     key,
		MaxPeers:       10,
		Discovery:      true,
		BootstrapNodes: boostNodes,
		Name:           name,
		Protocols:      []server.Protocol{pro},
		NAT:            nat.Any(),
		ListenAddr:     addr,
	}

	fmt.Println("Starting peer...")
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start peer: %v.\n", err)
		os.Exit(1)
	}

	return &server
}

// SendSelf wraps a payload into a Whisper envelope and forwards it to itself.
func selfSend(shh1, shh2 *whisper.Whisper, payload []byte) error {
	ok := make(chan struct{})

	// Start watching for self messages, output any arrivals
	id1 := shh1.NewIdentity()
	id2 := shh2.NewIdentity()
	shh1.Watch(whisper.Filter{
		To: &id1.PublicKey,
		Fn: func(msg *whisper.Message) {
			fmt.Printf("Message received: %s, signed with 0x%x.\n", string(msg.Payload), msg.Signature)
			close(ok)
		},
	})
	// Wrap the payload and encrypt it
	msg := whisper.NewMessage(payload)
	envelope, err := msg.Wrap(whisper.DefaultPoW, whisper.Options{
		From: id2,
		To:   &id1.PublicKey,
		TTL:  whisper.DefaultTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to seal message: %v", err)
	}
	// Dump the message into the system and wait for it to pop back out
	if err := shh2.Send(envelope); err != nil {
		return fmt.Errorf("failed to send self-message: %v", err)
	}
	select {
	case <-ok:
	case <-time.After(time.Second * 100):
		return fmt.Errorf("failed to receive message in time")
	}
	return nil
}
