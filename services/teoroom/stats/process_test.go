// Copyright 2019 Teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stats

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gocql/gocql"
	"github.com/kirill-scherba/teonet-go/services/teoroomcli/stats"
	"github.com/kirill-scherba/teonet-go/teonet/teonet"
)

type Teoemu struct{}

var answerData []byte

func (t *Teoemu) SendTo(peer string, cmd byte, data []byte) (int, error) { return 0, nil }
func (t *Teoemu) SendAnswer(pac interface{}, cmd byte, data []byte) (int, error) {
	answerData = data
	return 0, nil
}

func TestProcess_ComRoomCreated(t *testing.T) {

	teoemu := &Teoemu{}
	teo := &teonet.Teonet{}
	var err error
	var r *Rooms

	t.Run("Connect", func(t *testing.T) {
		r, err = Connect(teoemu, "teoroom_test")
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("Connected to database\n")
	})
	defer r.Close()

	t.Run("ComRoomCreated", func(t *testing.T) {
		// Create request and process it
		req := &stats.RoomCreateRequest{RoomID: gocql.TimeUUID(), RoomNum: 123}
		data, err := req.MarshalBinary()
		if err != nil {
			t.Error(err)
			return
		}
		pac := teo.PacketCreateNew("teo-from", 129, data)
		err = r.ComRoomCreated(pac)
		if err != nil {
			t.Error(err)
			return
		}
		// Check responce
		res := &stats.RoomCreateResponce{}
		err = res.UnmarshalBinary(answerData)
		if err != nil {
			t.Error(err)
			return
		}
		if res.RoomID.String() != req.RoomID.String() {
			t.Error(errors.New("roomID in teonet answer does not equal to " +
				"generated roomID in ComRoomCreated function"))
		}
	})
}