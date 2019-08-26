package teonet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/kirill-scherba/net-example-go/teocli/teocli"
	"github.com/kirill-scherba/net-example-go/teolog/teolog"
	"github.com/kirill-scherba/net-example-go/trudp/trudp"
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
	mux   sync.Mutex         // Maps mutex
}

// packet is Packet processing channels data structure
type packet struct {
	packet []byte
	client *client
}

// \TDODO: use this interface to make one parameter from 'conn net.Conn' and
// '*trudp.ChannelData'
type conn interface {
	Write()
	Close()
}

// client is clients data structure
type client struct {
	name string // name
	addr string // address (ip:port:ch)
	//tcp  bool               // connection type: tcp - if true, trudp - if false
	conn net.Conn           // tcp connection
	tcd  *trudp.ChannelData // trudp connection
	cli  *teocli.TeoLNull   // teocli connection to use readBuffer
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
	l0.mux.Lock()
	l0.ma[client.addr] = client
	l0.mn[client.name] = client
	l0.mux.Unlock()
}

// close closes(disconnect) connected client
func (l0 *l0) close(client *client) {
	teolog.Debugf(MODULE, "client %s (%s) disconnected\n", client.name, client.addr)
	if client.conn != nil {
		client.conn.Close()
	}
	l0.mux.Lock()
	delete(l0.ma, client.addr)
	delete(l0.mn, client.name)
	l0.mux.Unlock()
}

// closeAddr closes(disconnect) connected client by address
func (l0 *l0) closeAddr(addr string) bool {
	client, ok := l0.find(addr)
	if ok {
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
				teolog.Debug(MODULE, "stop accepting: ", err.Error())
				//os.Exit(1)
				break
			}
			// Handle connections in a new goroutine.
			go l0.handleConnection(conn)
		}
		teolog.Connect(MODULE, "l0 server stop listen tcp port:", port)
	}(*port)
}

// find finds client in clients map by address
func (l0 *l0) find(addr string) (client *client, ok bool) {
	l0.mux.Lock()
	client, ok = l0.ma[addr]
	l0.mux.Unlock()
	return
}

// toprocess send packet to packet processing
func (l0 *l0) toprocess(p []byte, cli *teocli.TeoLNull, addr string, i ...interface{}) {
	if len(i) > 0 {
		pac := &packet{packet: p, client: &client{cli: cli, addr: addr}}
		switch conn := i[0].(type) {
		case net.Conn:
			//pac.client.tcp = true
			pac.client.conn = conn
		case *trudp.ChannelData:
			//pac.client.tcp = false
			pac.client.tcd = conn
			//fmt.Printf("%v\n", pac.client.conn == nil)
		}
		l0.ch <- pac
		return
	}
}

// check checks that received trudp packet is l0 packet and process it so.
// Return satatus 1 if not processed(if it is not teocli packet), or 0 if
// processed and send, or -1 if part of packet received and we wait next
// subpacket
func (l0 *l0) check(tcd *trudp.ChannelData, packet []byte) (p []byte, status int) {
	// Find this trudp key (connection) in cliens table and get cli, or
	// create new if not fount
	var cli *teocli.TeoLNull
	key := tcd.GetKey()
	client, ok := l0.find(key)
	if !ok {
		cli, _ = teocli.Init(false) // create new client
	} else {
		cli = client.cli
	}
	return l0.packetCheck(cli, key, tcd, packet)
}

// packetCheck check that received tcp packet is l0 packet and process it so.
// Return satatus 1 if not processed(if it is not teocli packet), or 0 if
// processed and send, or -1 if part of packet received and we wait next subpacket
func (l0 *l0) packetCheck(cli *teocli.TeoLNull, addr string, conn interface{}, data []byte) (p []byte, status int) {
check:
	p, status = cli.PacketCheck(data)
	switch status {
	case 0:
		l0.toprocess(p, cli, addr, conn)
		data = nil
		goto check // check next packet in read buffer
	case -1:
		if data != nil {
			teolog.Debugf(MODULE, "packet not received yet (got part of packet)\n")
		}
	case 1:
		teolog.Debugf(MODULE, "wrong packet received (drop it): %d, data: %v\n", len(p), p)
	}
	return
}

// Handle TCP connection
func (l0 *l0) handleConnection(conn net.Conn) {
	teolog.Connectf(MODULE, "l0 server tcp client %v connected...", conn.RemoteAddr())
	cli, _ := teocli.Init(true)
	b := make([]byte, 2048)
	for {
		n, err := conn.Read(b)
		if err != nil {
			break
		}
		teolog.Debugf(MODULE, "got %d bytes data from tcp clien: %v\n",
			n, conn.RemoteAddr().String())
		l0.packetCheck(cli, conn.RemoteAddr().String(), conn, b[:n])
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
			teolog.Debugf(MODULE,
				"valid packet received from client %s, length: %d\n",
				pac.client.addr, len(pac.packet))

			p := pac.client.cli.PacketNew(pac.packet)

			// Find address in clients map and add if it absent, or send packet to
			// peer if client already exsists
			client, ok := l0.find(pac.client.addr)
			if !ok {
				if p.Command() == 0 && p.Name() == "" {
					pac.client.name = string(p.Data())
					l0.add(pac.client)
				} else {
					teolog.Debugf(MODULE,
						"incorrect login packet received from client %s, disconnect...\n",
						pac.client.addr)
					// pac.client.conn.Write([]byte("HTTP/1.1 200 OK\n" +
					// 	"Content-Type: text/html\n\n" +
					// 	"<html><body>Hello!</body></html>\n"))
					pac.client.conn.Close()
				}
				continue
			} else {
				l0.sendToPeer(client.name, p.Command(), p.Name(), p.Data())
			}
		}
		l0.closeAll()
		teolog.Debugf(MODULE, "l0 packet process stopped\n")
		l0.teo.wg.Done()
	}()
}

// sendToPeer from L0 server, send clients packet received from client to peer
func (l0 *l0) sendToPeer(from string, cmd int, peer string, data []byte) {
	teolog.Debugf(MODULE,
		"send cmd: %d, %d bytes data packet to peer %s, from client: %s",
		cmd, len(data), peer, from)

	buf := new(bytes.Buffer)
	le := binary.LittleEndian
	binary.Write(buf, le, byte(cmd))               // Command
	binary.Write(buf, le, byte(len(from)+1))       // From client name length (include leading zero)
	binary.Write(buf, le, uint16(len(data)))       // Packet data length
	binary.Write(buf, le, append([]byte(from), 0)) // From client name (include leading zero)
	binary.Write(buf, le, []byte(data))            // Packet data
	l0.teo.SendTo(peer, CmdL0, buf.Bytes())        // Send to peer
}

// cmdL0 parse cmd got from L0 server with packet from L0 client
func (l0 *l0) cmdL0(rec *receiveData) {
	l0.teo.com.log(rec.rd, "CMD_L0 command")
}

// cmdL0To parse cmd got from peer to L0 client
func (l0 *l0) cmdL0To(rec *receiveData) {
	l0.teo.com.log(rec.rd, "CMD_L0_TO command")
	if !l0.allow {
		teolog.Debugf(MODULE, "can't process this command because I'm not L0 server\n")
		return
	}

	// Parse command data
	buf := bytes.NewReader(rec.rd.Data())
	le := binary.LittleEndian
	var cmd, fromLen byte
	var dataLen uint16
	binary.Read(buf, le, &cmd)
	binary.Read(buf, le, &fromLen)
	binary.Read(buf, le, &dataLen)
	name := make([]byte, fromLen)
	binary.Read(buf, le, name)
	data := make([]byte, dataLen)
	binary.Read(buf, le, data)
	from := rec.rd.From()

	// Get client data from name map
	l0.mux.Lock()
	client, ok := l0.mn[string(name[:len(name)-1])]
	l0.mux.Unlock()
	if !ok {
		teolog.Debugf(MODULE, "can't find client %s in clients map\n", name)
		return
	}

	teolog.Debugf(MODULE, "got cmd: %d, %d bytes data packet from peer %s, to client: %s\n",
		cmd, dataLen, from, name)

	packet, err := client.cli.PacketCreate(uint8(cmd), from, data)
	if err != nil {
		teolog.Error(MODULE, err.Error())
		return
	}
	if client.conn != nil {
		teolog.Debugf(MODULE, "send cmd: %d, %d bytes data packet, to tcp L0 client: %s\n",
			cmd, dataLen, client.name)
		client.conn.Write(packet)
	} else {
		teolog.Debugf(MODULE, "send cmd: %d, %d bytes data packet, to trudp L0 client: %s\n", cmd,
			dataLen, client.name)
		client.tcd.Write(packet)
	}
}
