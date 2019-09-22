// Copyright 2019 teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Teonet room controller application.
// Teo room unites users to room and send commands between it

package main

import (
	"fmt"

	"github.com/kirill-scherba/teonet-go/services/teoroom"
	"github.com/kirill-scherba/teonet-go/teonet/teonet"
)

func main() {

	// Version this teonet application version
	const Version = "0.0.1"

	// Teonet logo
	teonet.Logo("Teonet-go room conroller service", Version)

	// Read Teonet parameters from configuration file and parse application
	// flars and arguments
	param := teonet.Params()

	// Show host and network name
	fmt.Printf("\nhost: %s\nnetwork: %s\n", param.Name, param.Network)

	// Start room controller
	tr, err := teoroom.Init()
	if err != nil {
		panic(err)
	}
	defer tr.Destroy()

	// Teonet connect and run
	teo := teonet.Connect(param, []string{"teo-go", "teo-room"}, Version)
	teo.Run(func(teo *teonet.Teonet) {
		for ev := range teo.Event() {

			// Event processing
			switch ev.Event {

			// When teonet started
			case teonet.EventStarted:
				fmt.Printf("Event Started\n")
			// case teonet.EventStoppedBefore:
			// case teonet.EventStopped:
			// 	fmt.Printf("Event Stopped\n")

			// When teonet peer connected
			case teonet.EventConnected:
				pac := ev.Data
				fmt.Printf("Event Connected from: %s\n", pac.From())

			// When teonet peer connected
			case teonet.EventDisconnected:
				pac := ev.Data
				fmt.Printf("Event Disconnected from: %s\n", pac.From())

			// When received command from teonet peer or client
			case teonet.EventReceived:
				pac := ev.Data
				fmt.Printf("Event Received from: %s, cmd: %d, data: %v\n",
					pac.From(), pac.Cmd(), pac.Data())

				// Commands processing
				switch pac.Cmd() {

				// Command #129: [in,out] Room request
				case teoroom.ComRoomRequest:
					if err := tr.RoomRequest(pac.From()); err != nil {
						fmt.Printf("Client %s is already connected\n", pac.From())
						break
					}
					// Send roomDataCommand
					teo.SendToClient("teo-l0", pac.From(), teoroom.ComRoomRequestAnswer, pac.Data())
					// Send all connected clients data to this new
					// \TODO replace sleep for normal protocol exchange:
					// - send in room request his number (position) in room
					// - wait while loadded and send his position
					// - and than send him position of already loadded users
					// go func() {
					// 	time.Sleep(500 * time.Millisecond)
					// 	tr.NewClient(pac.From(), func(l0, client string, data []byte) {
					// 		//teoroom.SendData(teo, client, pac.From(), data)
					// 		d := append(data, []byte(client)...)
					// 		fmt.Printf("send to %s data: %v\n", pac.From(), d)
					// 		teo.SendToClient("teo-l0", pac.From(), teoroom.ComRoomData, d)
					// 	})
					// }()

				// Command #130: [in,out] Data transfer
				case teoroom.ComRoomData:
					tr.GotData(pac.From(), pac.Data(), func(l0, client string, data []byte) {
						if data == nil {
							data = pac.Data()
						}
						//teoroom.SendData(teo, client, pac.From(), data)
						teo.SendToClient("teo-l0", client, teoroom.ComRoomData, append(data, []byte(pac.From())...))
					})

				// Command #131 [in] Disconnect (exit) from room
				case teoroom.ComDisconnect:
					if err := tr.Disconnect(pac.From()); err != nil {
						fmt.Printf("Client %s is already connected\n", pac.From())
						break
					}
				}
			}
		}
	})
}