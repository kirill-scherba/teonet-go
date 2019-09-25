// Copyright 2019 teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Teonet client using linux terminal game engine
//
// This is simple terminal game with teonet client connected to teonet l0
// server and teonet room controller. This application connect to l0 server
// first and than request room in room controller. When room controller answer
// with room request answer this game application can send its hero position and
// show position of other players entered to the same room.
//
// Install client and server:
//
//  go get github.com/kirill-scherba/teonet-go/teocli/examples/teocli-termloop
//  go get github.com/kirill-scherba/teonet-go/teonet
//
// Run server applications:
//
//  # run teonet l0 server
//  cd $GOPATH/src/github.com/kirill-scherba/teonet-go/teonet
//  go run . -p 7050 -l0-allow teo-l0
//
//  # run teonet room controller
//  cd $GOPATH/src/github.com/kirill-scherba/teonet-go/teonet/app/teoroom
//  go run . -r 7050 teo-room
//
// Run this game client application:
//
//  cd $GOPATH/src/github.com/kirill-scherba/teonet-go/teocli/examples/teocli-termloop
//  go run . -r 7050 -peer teo-room -n game-01
//
//  cd $GOPATH/src/github.com/kirill-scherba/teonet-go/teocli/examples/teocli-termloop
//  go run . -r 7050 -peer teo-room -n game-02
//
// To exit from this game type Ctrl+C twice. When you start play next time
// you'll be connected to another room.
//
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	tl "github.com/JoelOtter/termloop"
	"github.com/kirill-scherba/teonet-go/teocli/teocli"
)

// Version this teonet application version
const Version = "0.0.1"

// Teogame this game data structure
type Teogame struct {
	game   *tl.Game               // Game
	level  []*tl.BaseLevel        // Game levels
	hero   *Hero                  // Game Hero
	player map[byte]*Player       // Game Players map
	teo    *teocli.TeoLNull       // Teonet connetor
	peer   string                 // Teonet room controller peer name
	com    *outputCommands        // Teonet output commands receiver
	rra    *roomRequestAnswerData // Room request answer data
}

// Player stucture of player
type Player struct {
	*tl.Entity
	prevX int
	prevY int
	level *tl.BaseLevel
	tg    *Teogame
}

// Hero struct of hero
type Hero struct {
	Player
}

// Text of text
type Text struct {
	*tl.Text
	i int
}

// main parse aplication parameters and connect to Teonet. When teonet connected
// the game started
func main() {
	fmt.Println("Teocli termloop application ver " + Version)

	// Flags variables
	var name string  // this client name
	var peer string  // room controller peer name
	var raddr string // l0 server address
	var rport int    // l0 server port
	var timeout int  // reconnect timeout (in seconds)
	var tcp bool     // connect by TCP flag

	// Flags
	flag.StringVar(&name, "n", "teocli-go-main-test-01", "this application name")
	flag.StringVar(&peer, "peer", "teo-room", "teo-room peer name (to send commands to)")
	flag.StringVar(&raddr, "a", "localhost", "remote host address (to connect to remote host)")
	flag.IntVar(&rport, "r", 9010, "l0 server port (to connect to l0 server)")
	flag.BoolVar(&tcp, "tcp", false, "connect by TCP")
	flag.IntVar(&timeout, "t", 5, "reconnect after timeout (in second)")
	flag.Parse()

	// Run teonet game (connect to teonet, start game and process received commands)
	run(name, peer, raddr, rport, tcp, time.Duration(timeout)*time.Second)
}

// Run connect to teonet, start game and process received commands
func run(name, peer, raddr string, rport int, tcp bool, timeout time.Duration) (tg *Teogame) {
	tg = &Teogame{peer: peer, player: make(map[byte]*Player)}
	teocli.Run(name, raddr, rport, tcp, timeout, startCommand(tg), inputCommands(tg)...)
	return
}

// startGame initialize and start game
func (tg *Teogame) startGame(rra *roomRequestAnswerData) {
	tg.game = tl.NewGame()
	tg.game.Screen().SetFps(30)
	tg.rra = rra

	// Base level
	level := tl.NewBaseLevel(tl.Cell{
		Bg: tl.ColorBlack,
		Fg: tl.ColorWhite,
		Ch: ' ',
	})
	tg.level = append(tg.level, level) // Level 0: Game

	// Lake
	level.AddEntity(tl.NewRectangle(10, 5, 10, 5, tl.ColorWhite|tl.ColorBlack))

	// Text
	level.AddEntity(&Text{tl.NewText(0, 0, os.Args[0], tl.ColorBlack, tl.ColorBlue), 0})

	// Hero
	tg.hero = tg.addHero(int(rra.clientID)*3, 2)

	// Level 1: Game over
	tg.level = append(tg.level, func() (level *tl.BaseLevel) {
		level = tl.NewBaseLevel(tl.Cell{
			Bg: tl.ColorBlack,
			Fg: tl.ColorWhite,
			Ch: '*',
		})
		level.AddEntity(newGameOverText(tg))
		return
	}())

	// Start and run
	tg.game.Screen().SetLevel(tg.level[0])
	_, err := tg.com.sendData(tg.hero)
	if err != nil {
		panic(err)
	}
	tg.game.Start()

	// When stopped (press exit from game or Ctrl+C)
	fmt.Printf("game stopped\n")
	tg.com.disconnect()
	tg.com.stop()
}

// gameOver switch to 'game over' screen
func (tg *Teogame) gameOver() {
	fmt.Printf("Game over!\n")
	tg.game.Screen().SetLevel(tg.level[1])
}

// startGame reset game, request new game and switch to 'game' screen
func (tg *Teogame) startNewGame() {
	//tg.resetGame()
	tg.com.stcom.Command(tg.teo, nil)
	fmt.Printf("Start new game!\n")
	//tg.game.Screen().SetLevel(tg.level[0])
}

// resetGame reset game to it default values
func (tg *Teogame) resetGame() {
	for _, p := range tg.player {
		tg.level[0].RemoveEntity(p)
	}
	tg.player = make(map[byte]*Player)
	tg.game.Screen().SetLevel(tg.level[0])
}

// addHero add Hero to game
func (tg *Teogame) addHero(x, y int) (hero *Hero) {
	hero = &Hero{Player{
		Entity: tl.NewEntity(1, 1, 1, 1),
		level:  tg.level[0],
		tg:     tg,
	}}
	// Set the character at position (0, 0) on the entity.
	hero.SetCell(0, 0, &tl.Cell{Fg: tl.ColorGreen, Ch: 'Ω'})
	hero.SetPosition(x, y)
	tg.level[0].AddEntity(hero)
	return
}

// addPlayer add new Player to game or return existing if already exist
func (tg *Teogame) addPlayer(id byte) (player *Player) {
	player, ok := tg.player[id]
	if !ok {
		player = &Player{
			Entity: tl.NewEntity(2, 2, 1, 1),
			level:  tg.level[0],
			tg:     tg,
		}
		// Set the character at position (0, 0) on the entity.
		player.SetCell(0, 0, &tl.Cell{Fg: tl.ColorBlue, Ch: 'Ö'})
		tg.level[0].AddEntity(player)
		tg.player[id] = player
	}
	return
}

// Set player at center of map
// func (player *Player) Draw(screen *tl.Screen) {
// 	screenWidth, screenHeight := screen.Size()
// 	x, y := player.Position()
// 	player.level.SetOffset(screenWidth/2-x, screenHeight/2-y)
// 	player.Entity.Draw(screen)
// }

// Tick frame tick
func (player *Hero) Tick(event tl.Event) {
	if event.Type == tl.EventKey { // Is it a keyboard event?
		player.prevX, player.prevY = player.Position()

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

		// Check position changed and send to Teonet if so
		x, y := player.Position()
		if x != player.prevX || y != player.prevY {
			_, err := player.tg.com.sendData(player)
			if err != nil {
				panic(err)
			}
		}
	}
}

// Collide check colliding
func (player *Player) Collide(collision tl.Physical) {
	// Check if it's a Rectangle we're colliding with
	if _, ok := collision.(*tl.Rectangle); ok {
		player.SetPosition(player.prevX, player.prevY)
		_, err := player.tg.com.sendData(player)
		if err != nil {
			panic(err)
		}
	}
}

// MarshalBinary marshal players data to binary
func (player *Player) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)
	x, y := player.Position()
	binary.Write(buf, binary.LittleEndian, player.tg.rra.clientID)
	if err = binary.Write(buf, binary.LittleEndian, int64(x)); err != nil {
		return
	} else if err = binary.Write(buf, binary.LittleEndian, int64(y)); err != nil {
		return
	}
	data = buf.Bytes()
	return
}

// UnmarshalBinary unmarshal binary data and sen it yo player
func (player *Player) UnmarshalBinary(data []byte) (err error) {
	var cliID byte
	var x, y int64
	buf := bytes.NewReader(data)
	if err = binary.Read(buf, binary.LittleEndian, &cliID); err != nil {
		return
	} else if err = binary.Read(buf, binary.LittleEndian, &x); err != nil {
		return
	} else if err = binary.Read(buf, binary.LittleEndian, &y); err != nil {
		return
	}
	player.SetPosition(int(x), int(y))
	return
}

// Tick of Text object
func (m *Text) Tick(ev tl.Event) {
	m.i++
	m.Text.SetText(os.Args[0] + ", frame: " + strconv.Itoa(m.i))
}

// newGameOverText create GameOverText object
func newGameOverText(tg *Teogame) (text *GameOverText) {
	t := []*tl.Text{}
	t = append(t, tl.NewText(0, 0, " Game over! ", tl.ColorBlack, tl.ColorBlue))
	t = append(t, tl.NewText(0, 0, " press any key to continue ",
		tl.ColorDefault, tl.ColorDefault))
	text = &GameOverText{t, tg}
	return
}

// GameOverText is type of text
type GameOverText struct {
	t  []*tl.Text
	tg *Teogame
}

// Draw game over text
func (got *GameOverText) Draw(screen *tl.Screen) {
	screenWidth, screenHeight := screen.Size()
	for i, t := range got.t {
		width, height := t.Size()
		t.SetPosition((screenWidth-width)/2, i*2+(screenHeight-height)/2)
		t.Draw(screen)
	}
}

// Tick check key pressed and start new game or quit
func (got *GameOverText) Tick(event tl.Event) {
	if event.Type == tl.EventKey { // Is it a keyboard event?
		// switch event.Ch { // If so, switch on the pressed key.
		// case 'p':
		got.tg.startNewGame()
		// }
	}
}