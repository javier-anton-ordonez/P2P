package tutorial

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	//"github.com/multiformats/go-multiaddr"
)

const protocolID = "/example/1.0.0"
const discoveryNamespace = "example"

type discoveryNotifee struct {
	host host.Host
}
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {

	err := n.host.Connect(context.Background(), pi)
	if err != nil {
		return
	}

	// Solo el peer con el ID "menor" abre el stream.
	if n.host.ID().String() < pi.ID.String() {

		fmt.Println("Opening stream")

		s, err := n.host.NewStream(
			context.Background(),
			pi.ID,
			protocolID,
		)
		if err != nil {
			return
		}

		go writeCounter(s)
		go readCounter(s)
	}
}
/*func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Println("Found:", pi.ID)

	if err := n.host.Connect(context.Background(), pi); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Connected!")

	s, err := n.host.NewStream(
		context.Background(),
		pi.ID,
		protocolID,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	go writeCounter(s)
	go readCounter(s)
}*/

func writeCounter(s network.Stream) {
	var counter uint64

	for {
		<-time.After(time.Second)
		counter++

		err := binary.Write(s, binary.BigEndian, counter)

		if err != nil {
			panic(err)
		}
	}
}

func readCounter(s network.Stream) {
	for {
		var counter uint64

		err := binary.Read(s, binary.BigEndian, &counter)

		if err != nil {
			panic(err)
		}
		fmt.Printf("Recibed number: %d from %s\n", counter, s.ID())
	}
}

func main() {
	// Add -peer-address flag
	//peerAddr := flag.String("peer-address", "", "peer address")
	flag.Parse()

	// Create the libp2p host.
	//
	// Note that we are explicitly passing the listen address and restricting it to IPv4 over the
	// loopback interface (127.0.0.1).
	//
	// Setting the TCP port as 0 makes libp2p choose an available port for us.
	// You could, of course, specify one if you like.
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		panic(err)
	}
	defer host.Close()

	// Print this node's addresses and ID
	fmt.Println("Addresses:", host.Addrs())
	fmt.Println("ID:", host.ID())

	host.SetStreamHandler(protocolID, func(s network.Stream) {
		go writeCounter(s)
		go readCounter(s)
	})

	service := mdns.NewMdnsService(
		host,
		"example-libp2p",
		&discoveryNotifee{host: host},
	)

	if err := service.Start(); err != nil {
		panic(err)
	}
	defer service.Close()

	/*// If we received a peer address, we should connect to it.
	if *peerAddr != "" {
		// Parse the multiaddr string.
		peerMA, err := multiaddr.NewMultiaddr(*peerAddr)
		if err != nil {
			panic(err)
		}
		peerAddrInfo, err := peer.AddrInfoFromP2pAddr(peerMA)
		if err != nil {
			panic(err)
		}

		// Connect to the node at the given address.
		if err := host.Connect(context.Background(), *peerAddrInfo); err != nil {
			panic(err)
		}
		fmt.Println("Connected to", peerAddrInfo.String())

		s, err := host.NewStream(
			context.Background(),
			peerAddrInfo.ID,
			protocolID,
		)
		if err != nil {
			panic(err)
		}

		go writeCounter(s)
		go readCounter(s)
		}*/

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh
}
