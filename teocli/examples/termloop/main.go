// Copyright 2019 teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Teonet client using termloop game engine

package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"strings"
	"time"

	tl "github.com/JoelOtter/termloop"
	"github.com/kirill-scherba/teonet-go/services/teoroom"
	"github.com/kirill-scherba/teonet-go/teocli/teocli"
)

// Version this teonet application version
const Version = "0.0.1"

// Teogame this game data structure
type Teogame struct {
	teo       *teocli.TeoLNull // teonet connetor
	com       *Commands        // teonet commands
	peer      string           // teonet room controller peer name
	connected bool             // is connected to teonet
	started   bool             // is game started
}

// Commands this game teonet commands receiver
type Commands struct {
	tg *Teogame
}

// Player data stucture
type Player struct {
	*tl.Entity
	prevX int
	prevY int
	level *tl.BaseLevel
	tg    *Teogame
}

type Hero struct {
	Player
}

// main parse aplication parameters and connect to Teonet. When teonet connected
// the game started
func main() {
	fmt.Println("Teocli termloop application ver " + Version)

	// Flags variables
	var name string      // this client name
	var peer string      // remote server name (to send commands to)
	var raddr string     // remote host address
	var rport, rchan int // remote host port and channel (for TRUDP)
	var timeout int      // send echo timeout (in microsecond)
	var tcp bool         // connect by TCP flag

	// Flags
	flag.StringVar(&name, "n", "teocli-go-main-test-01", "this application name")
	flag.StringVar(&peer, "peer", "teo-room", "teo-room peer name (to send commands to)")
	flag.StringVar(&raddr, "a", "localhost", "remote host address (to connect to remote host)")
	flag.IntVar(&rchan, "c", 0, "remote host channel (to connect to remote host TRUDP channel)")
	flag.IntVar(&rport, "r", 9010, "remote host port (to connect to remote host)")
	flag.IntVar(&timeout, "t", 1000000, "send echo timeout (in microsecond)")
	flag.BoolVar(&tcp, "tcp", false, "connect by TCP")
	flag.Parse()

	// Run teonet (connect to teonet and process received commands)
	var tg *Teogame
	tg = &Teogame{peer: peer, com: &Commands{}}
	tg.com.tg = tg
	tg.connect(name, raddr, rport, tcp, 5*time.Second)
}

// network return string with type of network
func (tg *Teogame) network(tcp bool) string {
	if tcp {
		return "TCP"
	}
	return "TRUDP"
}

// roomRequest [out] send RoomRequest command to room controller
func (com *Commands) roomRequest() {
	//com.tg.teo.Send(129, com.tg.peer, nil)
	teoroom.RoomRequest(com.tg.teo, com.tg.peer, nil)
}

// roomRequestAnswer [in] process RoomRequestAnswer command received from room
// controller
func (com *Commands) roomRequestAnswer(packet *teocli.Packet) {
	if !com.tg.started {
		go com.tg.game()
		com.tg.started = true
	}
}

// sendData [out] send data command to room controller
func (com *Commands) sendData(i ...interface{}) error {
	return teoroom.SendData(com.tg.teo, com.tg.peer, i...)
}

// gotData [out] process data command received from room controller
func (com *Commands) gotData(packet *teocli.Packet) {

}

// connect Connect to Teonet and process received commands
func (tg *Teogame) connect(name, raddr string, rport int, tcp bool, reconnectAfter time.Duration) {

	var err error

	// Reconnect loop, reconnect if disconnected afer reconnectAfter time (in sec)
	for {
		// Connect to L0 server
		fmt.Printf("try %s connecting to %s:%d ...\n", tg.network(tcp), raddr, rport)
		tg.teo, err = teocli.Connect(raddr, rport, tcp)
		if err != nil {
			fmt.Println(err)
			time.Sleep(reconnectAfter)
			continue
		}
		tg.connected = true

		// Send Teonet L0 login (requered after connect)
		fmt.Printf("send login\n")
		if _, err := tg.teo.SendLogin(name); err != nil {
			panic(err)
		}

		// Send peers command (just for test, it may be removed)
		fmt.Printf("send peers request\n")
		tg.teo.SendTo(tg.peer, teocli.CmdLPeers, nil)

		// Send Start game request to the teo-room
		tg.com.roomRequest()

		// Reader (receive data and process it)
		for {
			packet, err := tg.teo.Read()
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Printf("got cmd %d from %s, data len: %d, data: %v\n",
				packet.Command(), packet.From(), len(packet.Data()), packet.Data())

			switch packet.Command() {

			// RoomRequestAnswer
			case teoroom.ComRoomRequestAnswer:
				tg.com.roomRequestAnswer(packet)

			// Echo answer
			case teocli.CmdLEchoAnswer:
				if t, err := packet.TripTime(); err != nil {
					fmt.Println("trip time error:", err)
				} else {
					fmt.Println("trip time (ms):", t)
				}

			// Peers answer (just for test, it may be removed)
			case teocli.CmdLPeersAnswer:
				ln := strings.Repeat("-", 59)
				fmt.Println("PeerAnswer received\n"+ln, "\n"+packet.Peers()+ln)
			}
		}

		// Disconnect
		tg.teo.Disconnect()
		tg.connected = false
		time.Sleep(reconnectAfter)
	}
}

// Run game
func (tg *Teogame) game() {
	game := tl.NewGame()
	game.Screen().SetFps(30)
	level := tl.NewBaseLevel(tl.Cell{
		Bg: tl.ColorBlack,
		Fg: tl.ColorWhite,
		Ch: ' ',
	})
	level.AddEntity(tl.NewRectangle(10, 10, 50, 20, tl.ColorBlue))

	// Hero
	player := Hero{Player{
		Entity: tl.NewEntity(1, 1, 1, 1),
		level:  level,
		tg:     tg,
	}}
	// Set the character at position (0, 0) on the entity.
	player.SetCell(0, 0, &tl.Cell{Fg: tl.ColorGreen, Ch: 'Ω'})
	level.AddEntity(&player)

	// Players
	player2 := Player{
		Entity: tl.NewEntity(2, 2, 1, 1),
		level:  level,
		tg:     tg,
	}
	// Set the character at position (0, 0) on the entity.
	player2.SetCell(0, 0, &tl.Cell{Fg: tl.ColorBlue, Ch: '∩'})
	level.AddEntity(&player2)

	game.Screen().SetLevel(level)
	game.Start()
	fmt.Printf("game stopped\n")
	tg.started = false
	//tg.teo.Disconnect()
}

// Set player at center of map
// func (player *Player) Draw(screen *tl.Screen) {
// 	screenWidth, screenHeight := screen.Size()
// 	x, y := player.Position()
// 	player.level.SetOffset(screenWidth/2-x, screenHeight/2-y)
// 	player.Entity.Draw(screen)
// }

func (player *Hero) Tick(event tl.Event) {
	if event.Type == tl.EventKey { // Is it a keyboard event?

		// Check position changed and send to Teonet if so
		x, y := player.Position()
		if x != player.prevX || y != player.prevY {
			err := player.tg.com.sendData(player)
			if err != nil {
				panic(err)
			}
		}
		player.prevX, player.prevY = x, y

		// Save previouse position and set to new position
		switch event.Key { // If so, switch on the pressed key.
		case tl.KeyArrowRight:
			player.SetPosition(player.prevX+1, player.prevY)
		case tl.KeyArrowLeft:
			player.SetPosition(player.prevX-1, player.prevY)
		case tl.KeyArrowUp:
			player.SetPosition(player.prevX, player.prevY-1)
		case tl.KeyArrowDown:
			player.SetPosition(player.prevX, player.prevY+1)
		}
	}
}

func (player *Hero) Collide(collision tl.Physical) {
	// Check if it's a Rectangle we're colliding with
	if _, ok := collision.(*tl.Rectangle); ok {
		player.SetPosition(player.prevX, player.prevY)
	}
}

func (player *Player) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)
	x, y := player.Position()
	err = binary.Write(buf, binary.LittleEndian, int64(x))
	err = binary.Write(buf, binary.LittleEndian, int64(y))
	data = buf.Bytes()
	return
}

func (player *Player) UnmarshalBinary(data []byte) (err error) {
	var x, y int64
	buf := bytes.NewReader(data)
	err = binary.Read(buf, binary.LittleEndian, &x)
	if err != nil {
		return
	}
	err = binary.Read(buf, binary.LittleEndian, &y)
	if err != nil {
		return
	}
	player.SetPosition(int(x), int(y))
	return
}
