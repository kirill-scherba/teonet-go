package teonet

import (
	"fmt"
	"net"
	"strconv"

	"github.com/kirill-scherba/net-example-go/teocli/teocli"
	"github.com/kirill-scherba/net-example-go/teolog/teolog"
)

// Teonet L0 server module

// l0 is Module data structure
type l0 struct {
	teo   *Teonet            // Pointer to Teonet
	allow bool               // Allow L0 Server
	port  int                // TCP port (if 0 - not allowed TCP)
	conn  net.Listener       // TCP listener connection
	ch    chan *packet       // Packet processing channel
	ma    map[string]*client // Clients address map
	mn    map[string]*client // Clients name map
}

// packet is Packet processing channels data structure
type packet struct {
	packet []byte
	client *client
}

// client is clients data structure
type client struct {
	name string
	tcp  bool
	addr string
	conn net.Conn
	cli  *teocli.TeoLNull
}

// l0New initialize l0 module
func (teo *Teonet) l0New() *l0 {
	l0 := &l0{teo: teo, allow: teo.param.L0allow, port: teo.param.L0tcpPort}
	if l0.allow {
		teolog.Connect(MODULE, "l0 server start listen udp port:", l0.teo.param.Port)
		l0.ma = make(map[string]*client)
		l0.mn = make(map[string]*client)
		l0.process()
		//if l0.port > 0 {
		l0.tspServer(&l0.port)
		teo.param.L0tcpPort = l0.port
		//}
	}
	return l0
}

// destroy destroys l0 module
func (l0 *l0) destroy() {
	if l0.allow {
		teolog.Connect(MODULE, "l0 server stop listen udp port:", l0.teo.param.Port)
		if l0.conn != nil {
			l0.conn.Close()
		}
		close(l0.ch)
	}
}

// add adds new client
func (l0 *l0) add(client *client) {
	teolog.Debugf(MODULE, "new client %s (%s) connected\n", client.name, client.addr)
	l0.ma[client.addr] = client
	l0.mn[client.name] = client
}

// close closes(disconnect) connected client
func (l0 *l0) close(client *client) {
	teolog.Debugf(MODULE, "client %s (%s) disconnected\n", client.name, client.addr)
	if client.tcp {
		client.conn.Close()
	}
	delete(l0.ma, client.addr)
	delete(l0.mn, client.name)
}

// closeAddr closes(disconnect) connected client by address
func (l0 *l0) closeAddr(addr string) bool {
	if client, ok := l0.ma[addr]; ok {
		l0.close(client)
		return true
	}
	return false
}

// closeAll close(disconnect) all connected clients
func (l0 *l0) closeAll() {
	for _, client := range l0.mn {
		l0.close(client)
	}
}

// tspServer TCP L0 server
func (l0 *l0) tspServer(port *int) {

	const (
		network  = "tcp"
		hostName = ""
	)

	var err error

	// Start tcp server
	for {
		l0.conn, err = net.Listen(network, hostName+":"+strconv.Itoa(*port))
		if err == nil {
			break
		}
		*port++
		fmt.Println("the", *port-1, "is busy, try next port:", *port)
	}
	// If input function parameter port was 0 than get it from connection for
	// future use
	if *port == 0 {
		*port = l0.conn.Addr().(*net.TCPAddr).Port
	}
	teolog.Connect(MODULE, "l0 server start listen tcp port:", *port)

	// Listen for an incoming connection
	go func(port int) {
		for {
			conn, err := l0.conn.Accept()
			if err != nil {
				fmt.Println("Error accepting: ", err.Error())
				//os.Exit(1)
				break
			}
			// Handle connections in a new goroutine.
			go l0.handleConnection(conn)
		}
		teolog.Connect(MODULE, "l0 server stop listen tcp port:", port)
	}(*port)
}

// Handle TCP connection
func (l0 *l0) handleConnection(conn net.Conn) {
	teolog.Connectf(MODULE, "l0 server tcp client %v connected...", conn.RemoteAddr())
	conn.Write([]byte("HTTP/1.1 200 OK\nContent-Type: text/html\n\n<html><body>Hello!</body></html>\n"))
	cli, _ := teocli.Init(true)
	b := make([]byte, 2048)
	for {
		n, err := conn.Read(b)
		if err != nil {
			break
		}
		teolog.Debugf(MODULE, "got %d bytes data from tcp clien: %v\n",
			n, conn.RemoteAddr().String())
		p, status := cli.PacketCheck(b[:n])
		switch status {
		case 0:
			l0.ch <- &packet{
				packet: p,
				client: &client{cli: cli, tcp: true, addr: conn.RemoteAddr().String(), conn: conn},
			}
		case -1:
			teolog.Debugf(MODULE, "packet not received yet (got part of packet)\n")
		case 1:
			teolog.Debugf(MODULE, "wrong packet received (drop it): %d, data: %v\n", len(p), p)
		}
	}
	teolog.Connectf(MODULE, "l0 server tcp client %v disconnected...", conn.RemoteAddr())
	if !l0.closeAddr(conn.RemoteAddr().String()) {
		conn.Close()
	}
}

// Process received packets
func (l0 *l0) process() {
	teolog.Debugf(MODULE, "l0 packet process started\n")
	l0.ch = make(chan *packet)
	l0.teo.wg.Add(1)
	go func() {
		for pac := range l0.ch {
			teolog.Debugf(MODULE, "valid packet received from %s, length: %d, data: %v\n",
				pac.client.addr, len(pac.packet), pac.packet)
			// Find address in clients map
			if _, ok := l0.ma[pac.client.addr]; !ok {
				p := pac.client.cli.PacketNew(pac.packet)
				if p.Command() == 0 && p.Name() == "" {
					pac.client.name = string(p.Data())
					l0.add(pac.client)
				} else {
					teolog.Debugf(MODULE, "incorrect login packet received from %s, disconnect...\n",
						pac.client.addr)
					pac.client.conn.Close()
				}
			}
		}
		l0.closeAll()
		teolog.Debugf(MODULE, "l0 packet process stopped\n")
		l0.teo.wg.Done()
	}()
}