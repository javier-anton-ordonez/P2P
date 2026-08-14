package tutorial_test
/*
package main

import (

	"fmt"
	 "github.com/libp2p/go-libp2p"

)

	func main(){
		host, err := libp2p.New()

		if err != nil{
			fmt.Println(err.Error())
			return
		}

}
*/

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
)

const protocolID = "/example/1.0.0"

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

		err := binary.Read(s, binary.BigEndian, counter)

		if err != nil {
			panic(err)
		}
		fmt.Println("Recibed number: %d from %s\n", counter, s.ID())
	}
}

func main2() {

	host, err := libp2p.New()
	if err != nil {
		panic(err)
	}
	defer host.Close()

	fmt.Println("ID:", host.ID())
	fmt.Println("Addrs:", host.Addrs())

	host.SetStreamHandler(protocolID, func(s network.Stream) {
		go writeCounter(s)
		go readCounter(s)
	})

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

}
