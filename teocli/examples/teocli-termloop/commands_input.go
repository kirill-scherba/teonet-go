package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/kirill-scherba/teonet-go/services/teoroomcli"
	"github.com/kirill-scherba/teonet-go/teocli/teocli"
)

// init runs when packet main initialize (before main calls).It set seed to
// random module to use unical random values
func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

// Input command receivers
type authAnswerCommand struct{ tg *Teogame }
type echoAnswerCommand struct{}
type peerAnswerCommand struct{}
type roomRequestAnswerCommand struct{ tg *Teogame }
type roomDataCommand struct{ tg *Teogame }
type roomStartCommand struct{ tg *Teogame }
type clientDisconnecCommand struct{ tg *Teogame }

// newInputCommands combine input commands to slice (to use in teocli.Run() function)
func newInputCommands(tg *Teogame) (com []teocli.Command) {
	com = append(com,
		authAnswerCommand{tg},
		echoAnswerCommand{},
		peerAnswerCommand{},
		roomRequestAnswerCommand{tg},
		roomDataCommand{tg},
		roomStartCommand{tg},
		clientDisconnecCommand{tg},
	)
	return
}

// Auth answer command methods
func (p authAnswerCommand) Cmd() byte { return 129 }
func (p authAnswerCommand) Command(packet *teocli.Packet) bool {
	if packet.From() != "" {
		return false
	}
	// fmt.Printf("got auth command answer from: '%s', data: %s\n", packet.From(),
	// 	string(packet.Data()))
	// TODO: Login sucessfully.Start game and save received cookies
	p.tg.param.Cookies = string(packet.Data())
	p.tg.conf.Write()
	return true
}

// Echo answer command methods
func (p echoAnswerCommand) Cmd() byte { return teocli.CmdLEchoAnswer }
func (p echoAnswerCommand) Command(packet *teocli.Packet) bool {
	if t, err := packet.Triptime(); err != nil {
		fmt.Println("trip time error:", err)
	} else {
		fmt.Println("trip time (ms):", t)
	}
	return true
}

// Peer answer command methods
func (p peerAnswerCommand) Cmd() byte { return teocli.CmdLPeersAnswer }
func (p peerAnswerCommand) Command(packet *teocli.Packet) bool {
	ln := strings.Repeat("-", 59)
	fmt.Println("PeerAnswer received\n"+ln, "\n"+packet.Peers()+ln)
	return true
}

// Room request answer command methods
func (p roomRequestAnswerCommand) Cmd() byte { return teoroomcli.ComRoomRequestAnswer }
func (p roomRequestAnswerCommand) Command(packet *teocli.Packet) bool {
	rra := roomRequestAnswerData{}
	rra.UnmarshalBinary(packet.Data())
	if p.tg.game == nil {
		go p.tg.start(&rra)
	} else {
		p.tg.reset(&rra)
	}
	return true
}

// Room data command methods
func (p roomDataCommand) Cmd() byte { return teoroomcli.ComRoomData }
func (p roomDataCommand) Command(packet *teocli.Packet) bool {
	id := packet.Data()[0]
	p.tg.addPlayer(p.tg.level[Game], id).UnmarshalBinary(packet.Data())
	return true
}

// Disconnec (exit from room, game over) command methods
func (p clientDisconnecCommand) Cmd() byte { return teoroomcli.ComDisconnect }
func (p clientDisconnecCommand) Command(packet *teocli.Packet) bool {
	if packet.Data() == nil || len(packet.Data()) == 0 {
		p.tg.menu.SetText(" Game Over! ")
		p.tg.setLevel(Menu)
		return true
	}
	id := packet.Data()[0]
	if player, ok := p.tg.player[id]; ok {
		p.tg.level[0].RemoveEntity(player)
		delete(p.tg.player, id)
	}
	return true
}

// roomStartCommand start game command methods
func (p roomStartCommand) Cmd() byte { return teoroomcli.ComStart }
func (p roomStartCommand) Command(packet *teocli.Packet) bool {
	p.tg.state.setRunning()
	return true
}

// roomRequestAnswerData room request answer data structure
type roomRequestAnswerData struct {
	clientID byte
}

// MarshalBinary marshal roomRequestAnswerData to binary
func (rra *roomRequestAnswerData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.LittleEndian, rra.clientID); err != nil {
		return
	}
	data = buf.Bytes()
	return
}

// UnmarshalBinary unmarshal roomRequestAnswerData from binary
func (rra *roomRequestAnswerData) UnmarshalBinary(data []byte) (err error) {
	if data == nil || len(data) == 0 {
		rra.clientID = byte(rand.Intn(20))
		return
	}
	var x byte
	buf := bytes.NewReader(data)
	if err = binary.Read(buf, binary.LittleEndian, &x); err != nil {
		return
	}
	rra.clientID = x
	return
}
