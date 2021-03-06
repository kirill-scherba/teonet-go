package tunnel

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
)

// DefautPort is default port value for remote and local host
const (
	// DefaultInterface is default TAP device interface name
	DefaultInterface = "tap0"

	// DefaultMtu is default interface device mtu
	DefaultMtu = 1500

	// DefautPort is default port value for remote and local host
	DefautPort = 9669

	// DefaultMask is default ip adress mask
	DefaultMask = 24

	// Sleep time after udp socket error
	SleepErrTime = 100
)

// SetDefault set sefault parameters
func (p *Params) SetDefault() *Params {
	p.Intrface = DefaultInterface
	p.Mtu = DefaultMtu
	p.Rport = DefautPort
	p.Lport = DefautPort
	p.Lmask = DefaultMask
	return p
}

// Params is application parameters
type Params struct {
	Intrface  string // interface name
	Mtu       int    // interface mtu
	Laddr     string // interface local address
	Lmask     int    // interface local address mask
	Raddr     string // remote host address
	Rport     int    // remote host port
	Lport     int    // this host port
	ShowHelpF bool   // show help flag
}

// Tunnel define teonet tunnel data structure
type Tunnel struct {
	p     *Params          // Tunnel parameters
	iface *water.Interface // TAP interface
	sock  *net.UDPConn     // UDP socket
	raddr *net.UDPAddr     // Remote UDP address
	arp   *Arp             // Tunnel arp table
}

// New create new tunnel
func New(p *Params) (t *Tunnel) {
	t = &Tunnel{p: p}
	return t
}

// Run start and process teonet tunnel
func (t *Tunnel) Run() {
	t.newArp()
	t.newSocket()
	t.newInterface()
	go t.ifaceListner()
	t.udpListner()
}

// newInterface create new TAP interface, set ip address and up it
func (t *Tunnel) newInterface() {
	// Create a new tunnel device (requires root privileges).
	conf := water.Config{DeviceType: water.TAP}
	var err error
	t.iface, err = water.New(conf)
	if err != nil {
		log.Fatalf("error creating TAP device: %v", err)
	}

	// Setup IP properties.
	switch runtime.GOOS {
	case "linux":
		if err := exec.Command("/sbin/ip", "link", "set", "dev", t.iface.Name(),
			"mtu", strconv.Itoa(t.p.Mtu)).Run(); err != nil {
			log.Fatalf("ip link error: %v", err)
		}
		if err := exec.Command("/sbin/ip", "addr", "add",
			t.p.Laddr+"/"+strconv.Itoa(t.p.Lmask),
			"dev", t.iface.Name()).Run(); err != nil {
			log.Fatalf("ip addr error: %v", err)
		}
		if err := exec.Command("/sbin/ip", "link", "set", "dev", t.iface.Name(),
			"up").Run(); err != nil {
			log.Fatalf("ip link error: %v", err)
		}
	case "darwin":
		if err := exec.Command("/sbin/ifconfig", t.iface.Name(),
			strconv.Itoa(t.p.Mtu), strconv.Itoa(t.p.Mtu), t.p.Laddr, t.p.Raddr,
			"up").Run(); err != nil {
			log.Fatalf("ifconfig error: %v", err)
		}
	default:
		log.Fatalf("no TAP support for: %v", runtime.GOOS)
	}
	log.Printf("interface %s sucessfully created\n", t.iface.Name())
}

// newSocket create new UDP socket to listen, read and write UDP packets
func (t *Tunnel) newSocket() {
	// Create a new UDP socket
	laddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("", strconv.Itoa(t.p.Lport)))
	if err != nil {
		log.Fatalf("error resolving address: %v", err)
	}
	t.sock, err = net.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatalf("error listening on socket: %v", err)
	}
	log.Printf("UDP listner started at port %d\n", t.p.Lport)

	t.resolveRaddr()
}

// ifaceListner start iface listen and Handle outbound traffic
func (t *Tunnel) ifaceListner() {
	defer t.iface.Close()

	var num uint64
	var frame ethernet.Frame
	for {
		frame.Resize(t.p.Mtu)
		n, err := t.iface.Read([]byte(frame))
		if err != nil {
			// if isDone(ctx) {
			// 	return
			// }
			log.Fatalf("TAP read error: %v", err)
		}

		num++
		log.Printf("outbound frame (from iface) #%d:\n", num)
		fmt.Printf("read %d bytes packet from interface %s\n", n, t.iface.Name())
		if n == 0 {
			log.Printf("skip it\n\n")
			continue
		}

		// Show frame log
		f := frame[:n]
		fmt.Printf("Dst: %s\n", f.Destination())
		fmt.Printf("Src: %s\n", f.Source())
		fmt.Printf("Ethertype: % x (%s)\n", f.Ethertype(),
			Ethertype{f.Ethertype()}.String())
		fmt.Printf("Payload: % x\n", f.Payload())

		raddr, ok := t.arp.get(f.Destination())

		if !ok && f.Destination().String() == "ff:ff:ff:ff:ff:ff" {
			t.arp.foreach(func(haddr string, raddr *net.UDPAddr) {
				t.sock.WriteToUDP(frame[:n], raddr)
				fmt.Printf("send to %s\n", raddr)
				ok = true
			})
			if ok {
				continue
			}
		}

		if !ok {
			if t.raddr != nil {
				raddr = t.raddr
			} else {
				log.Printf("remote UDP connection does not established yet\n")
				continue
			}
		}
		if _, err := t.sock.WriteToUDP(frame[:n], raddr); err != nil {
			// if isDone(ctx) {
			// 	return
			// }
			log.Printf("net write error: %v", err)
			time.Sleep(SleepErrTime * time.Millisecond)
			continue
		}

		log.Printf("send %d bytes packet to UDP %s\n\n", n, t.raddr)
	}
}

// udpListner create new UDP socket, start listen UDP port and Handle inbound
// traffic
func (t *Tunnel) udpListner() {
	defer t.sock.Close()

	var num uint64
	var frame ethernet.Frame
	for {
		frame.Resize(t.p.Mtu)
		n, raddr, err := t.sock.ReadFromUDP([]byte(frame))
		if err != nil {
			// if isDone(ctx) {
			// 	return
			// }
			log.Printf("net read error: %v", err)
			time.Sleep(SleepErrTime * time.Millisecond)
			continue
		}

		num++
		log.Printf("inbound frame (to iface) #%d:\n", num)
		log.Printf("got %d bytes packet from UDP %s\n", n, raddr)
		if n == 0 {
			log.Printf("skip it\n\n")
			continue
		}

		// Show frame log
		f := frame[:n]
		fmt.Printf("Dst: %s\n", f.Destination())
		fmt.Printf("Src: %s\n", f.Source())
		fmt.Printf("Ethertype: % x (%s)\n", f.Ethertype(),
			Ethertype{f.Ethertype()}.String())
		fmt.Printf("Payload: % x\n", f.Payload())

		t.arp.set(f.Source(), raddr)
		fmt.Printf("%s", t.arp)
		// if t.raddr == nil {
		// 	t.raddr = raddr
		// }

		//raddr, ok := t.arp.get(f.Destination())

		// if !ok && f.Destination().String() == "ff:ff:ff:ff:ff:ff" {
		// 	t.arp.foreach(func(haddr string, raddr *net.UDPAddr) {
		// 		if haddr == f.Source().String() {
		// 			return
		// 		}
		// 		t.sock.WriteToUDP(frame[:n], raddr)
		// 		fmt.Printf("send to %s\n", raddr)
		// 		ok = true
		// 	})
		// 	if ok {
		// 		continue
		// 	}
		// }

		if _, err := t.iface.Write(frame[:n]); err != nil {
			// if isDone(ctx) {
			// 	return
			// }
			log.Fatalf("interface write error: %v", err)
		}
		log.Printf("write %d bytes packet to interface %s\n\n", n, t.iface.Name())
	}
}

// resolveRaddr create remote address if there is client connection and Raddr
// and Rport connection parameters are present
func (t *Tunnel) resolveRaddr() {
	if t.p.Raddr == "" || t.p.Rport <= 0 {
		return
	}

	raddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(t.p.Raddr,
		strconv.Itoa(t.p.Rport)))
	if err != nil {
		log.Fatalf("error resolving address: %v", err)
	}

	log.Printf("remote host address %s\n", raddr)
	t.raddr = raddr
}
