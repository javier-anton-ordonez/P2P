package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type config struct {
	port           int
	iptype         string
	isMain         bool
	directionsFile string
	manualConnect  string
	directionKey   string
}
type PeerInfo struct {
	PeerID    string   `json:"peerID"`
	PeerAddrs []string `json:"peerAddrs"`
}

type networkNotifiee struct {
	peers []PeerInfo
	host  host.Host
}

const knownpeersprotocol = "/loadknownpeers"

func (n *networkNotifiee) Listen(network.Network, multiaddr.Multiaddr) {}

func (n *networkNotifiee) ListenClose(network.Network, multiaddr.Multiaddr) {}

/*
func (n *networkNotifiee) Connected(

	net network.Network,
	conn network.Conn,

	) {
		peerID := conn.RemotePeer()

		addrs := net.Peerstore().Addrs(peerID)

		peerInfo := PeerInfo{
			peerID:    peerID.String(),
			peerAddrs: make([]string, 0),
		}

		for _, addr := range addrs {
			peerInfo.peerAddrs = append(peerInfo.peerAddrs, addr.String())
		}

		n.peers = append(n.peers, peerInfo)

		fmt.Println("Se ha conectado:", peerID)
		fmt.Println("Direcciones:")
		for _, addr := range addrs {
			fmt.Println("  ", addr)
		}
	}
*/
func (n *networkNotifiee) Connected(
	net network.Network,
	conn network.Conn,
) {
	peerID := conn.RemotePeer()
	addr := conn.RemoteMultiaddr()

	peerAddr := addr.Encapsulate(
		multiaddr.StringCast("/p2p/" + peerID.String()),
	)

	peerInfo := PeerInfo{
		PeerID: peerID.String(),
		PeerAddrs: []string{
			peerAddr.String(),
		},
	}

	n.peers = append(n.peers, peerInfo)
	fmt.Println(n.peers)
}

func (n *networkNotifiee) Disconnected(
	net network.Network,
	conn network.Conn,
) {
	log.Println("El nodo con ID", conn.RemotePeer(), "se ha desconectado")
}

func (n *networkNotifiee) OpenedStream(
	net network.Network,
	stream network.Stream,
) {
	fmt.Println("Esta abierto el stream con el peer", stream.ID())
}

func (n *networkNotifiee) ClosedStream(
	net network.Network,
	stream network.Stream,
) {
}
func (n *networkNotifiee) handleKnownPeersRequest(s network.Stream) {
	fmt.Println("He recibido un stream de", s.Conn().RemotePeer())

	data, err := json.Marshal(n.peers)
	if err != nil {
		log.Println("Error convirtiendo peers a JSON:", err)
		s.Close()
		return
	}

	_, err = s.Write(data)
	if err != nil {
		log.Println("Error enviando peers:", err)
		s.Close()
		return
	}

	s.Close()
}
func OpenNewStream(pi *peer.AddrInfo, n host.Host) {

	fmt.Println("Opening stream")

	s, err := n.NewStream(
		context.Background(),
		pi.ID,
		knownpeersprotocol,
	)
	if err != nil {
		return
	}
	go handleKnownPeers(s)
}

func loadOrCreatePrivateKey(fileName string) (crypto.PrivKey, error) {

	_, err := os.Stat(fileName)
	if err == nil {

		keyBytes, err := os.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		priv, err := crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}

		return priv, nil
	} else if errors.Is(err, os.ErrNotExist) {

		priv, _, err := crypto.GenerateKeyPair(
			crypto.Ed25519,
			-1,
		)
		if err != nil {
			log.Println("Error al generar la clave privada", err)
		}

		keyBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(fileName, keyBytes, 0600)
		if err != nil {
			log.Println("Error al escribir en el fichero")
			return nil, err
		}

		return priv, nil
	} else {
		fmt.Println("Ocurrió otro error (por ejemplo, de permisos):", err)

		return nil, err
	}
}

func handleKnownPeers(s network.Stream) {

	data, err := io.ReadAll(s)
	if err != nil {
		log.Println("Error leyendo peers:", err)
		return
	}

	var peers []PeerInfo

	err = json.Unmarshal(data, &peers)
	if err != nil {
		log.Println("Error convirtiendo JSON:", err)
		return
	}

	fmt.Println("Peers recibidos:")
	fmt.Println(peers)

	s.Close()
}

func connectMainPeer(mainpeer string, host host.Host) (bool, *peer.AddrInfo) {
	fmt.Println(mainpeer)
	ma, err := multiaddr.NewMultiaddr(mainpeer)
	if err != nil {
		log.Println("Error al convertir a multiaddr:", err)
		return false, nil
	}

	addrsInfo, err := peer.AddrInfoFromP2pAddr(ma)

	if err != nil {
		log.Println("Error al sacar la informacion del addres", err)
		return false, nil
	}

	err = host.Connect(context.Background(), *addrsInfo)
	if err != nil {
		log.Println("No se pudo conectar a", addrsInfo.ID)
		return false, nil
	}
	return true, addrsInfo
}

func loadKnownPeers(directionsFile string) ([]string, error) {

	file, err := os.Open(directionsFile)

	if err != nil {
		log.Fatal("No se ha podido abrir el fichero")
		return nil, err
	}

	defer file.Close()
	var peers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fmt.Println(scanner.Text())
		addrsAdd := scanner.Text()
		peers = append(peers, addrsAdd)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return peers, nil
}

func loadConfig() (*config, error) {

	manualConnect := flag.String("peer-2-connect" /*peer al que se conecta de base*/, "", "Main peer al que conectarse")
	isMain := flag.Bool("isMain", false, "Si el nodo es main o no")
	port := flag.Int("p", 1234, "Port to connect")
	directionKey := flag.String("keyFile", "identity.key", "Direcciones de la key privada")
	directionsFile := flag.String("difFile", "./direcciones.txt", "Direccion del fichero con direcciones de peers conocidos")
	ipType := flag.String("ipTyoe", "ip4", "Tipo de ip de conexion")
	flag.Parse()

	return &config{
		port:           *port,
		iptype:         *ipType,
		isMain:         *isMain,
		directionsFile: *directionsFile,
		manualConnect:  *manualConnect,
		directionKey:   *directionKey,
	}, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		return
	}
	priv, err := loadOrCreatePrivateKey(config.directionKey)
	if err != nil {
		log.Println(err)
		return
	}
	addrString := fmt.Sprintf("/%s/127.0.0.1/tcp/%d", config.iptype, config.port)

	host, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(addrString),
	)
	if err != nil {
		log.Println("Error al crear el nodo")
		return
	}
	defer host.Close()
	notifiee := &networkNotifiee{
		peers: make([]PeerInfo, 0),
		host:  host,
	}

	host.Network().Notify(notifiee)
	host.SetStreamHandler(
		knownpeersprotocol,
		notifiee.handleKnownPeersRequest,
	)
	defer host.Network().StopNotify(notifiee)
	addrs := host.Addrs()[0].String()
	id := host.ID().String()
	fmt.Println("########################################")
	fmt.Println(addrs)
	fmt.Println(host.ID())
	fmt.Printf("Connect: %s/p2p/%s", addrs, id)
	fmt.Println("\n########################################")
	if !config.isMain {
		//Codigo para que se conecte a main peer
		if config.manualConnect != "" {
			log.Println("Tiene un nodo asignado manualmente")
			connected, addrsInfo := connectMainPeer(config.manualConnect, host)
			if !connected {
				log.Println("No se pudo conectar al peer indicado manualmente")
			} else {
				go OpenNewStream(addrsInfo, host)
			}
		} else {
			log.Printf("Leyendo fichero de direcciones")
			peers, err := loadKnownPeers(config.directionsFile)
			if err != nil {
				log.Fatal("Error al leer el fichero de configuracion")
				return
			}
			for i, peer := range peers {
				fmt.Printf("Connecting peer N: %d", i)
				connected, addrsInfo := connectMainPeer(peer, host)
				if connected {
					go OpenNewStream(addrsInfo, host)
				}
			}
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	fmt.Println("Cerrando nodo...")
}
