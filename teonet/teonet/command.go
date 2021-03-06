// Copyright 2019 Teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Teonet command processing module.

package teonet

// #include "command.h"
// #include "packet.h"
import "C"

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
)

// Teonet commands
const (
	CmdNone               = C.CMD_NONE                // #00 Cmd none used as first peers command
	CmdConnectR           = C.CMD_CONNECT_R           // #04 A Peer want connect to r-host
	CmdConnect            = C.CMD_CONNECT             // #05 Inform peer about connected peer
	CmdDisconnect         = C.CMD_DISCONNECTED        // #06 Send to peers signal about disconnect
	CmdSplit              = C.CMD_SPLIT               // #68 Group of packets (Splited packets)
	CmdL0                 = C.CMD_L0                  // #70 Command from L0 Client
	CmdL0To               = C.CMD_L0_TO               // #71 Command to L0 Client
	CmdPeers              = C.CMD_PEERS               // #72 Get peers, allow JSON in request
	CmdPeersAnswer        = C.CMD_PEERS_ANSWER        // #73 Get peers answer
	cmdAuht               = C.CMD_AUTH                // #77 Auth command
	cmdAuthAnswer         = C.CMD_AUTH_ANSWER         // #78 Auth answer command
	CmdL0Clients          = C.CMD_L0_CLIENTS          // #79 Request clients list
	CmdL0ClientsAnswer    = C.CMD_L0_CLIENTS_ANSWER   // #80 Clients list
	CmdSubscribe          = C.CMD_SUBSCRIBE           // #81 Subscribe to event
	CmdUnsubscribe        = C.CMD_UNSUBSCRIBE         // #82 UnSubscribe from event
	CmdSubscribeAnswer    = C.CMD_SUBSCRIBE_ANSWER    // #83 Subscribe answer
	CmdL0ClientsNum       = C.CMD_L0_CLIENTS_N        // #84 Request clients number, allow JSON in request
	CmdL0ClientsNumAnswer = C.CMD_L0_CLIENTS_N_ANSWER // #85 Clients number
	CmdHostInfo           = C.CMD_HOST_INFO           // #90 Request host info, allow JSON in request
	CmdHostInfoAnswer     = C.CMD_HOST_INFO_ANSWER    // #91 Request host info, allow JSON in request
	CmdL0Auth             = C.CMD_L0_AUTH             // #96 L0 server auth request answer command
	CmdUser               = C.CMD_USER                // #129 User command
)

// JSON data prefix used in teonet requests
var JSON = []byte("JSON")

//var JSONs = []byte("\"JSON\"")

// BINARY data prefix used in teonet requests
var BINARY = []byte("BINARY")

// command commands module methods holder
type command struct {
	teo *Teonet
}

// process processed internal Teonet commands
func (com *command) process(rec *receiveData) (processed bool) {

	// For commands receiving from peer create new peer in art table
	if !rec.rd.IsL0() {
		com.teo.arp.peerNew(rec)
	}

	processed = true
	cmd := rec.rd.Cmd()

	// Process kernel commands
	switch cmd {

	case C.CMD_CONNECT_R:
		com.teo.rhost.cmdConnectR(rec)

	case C.CMD_NONE, C.CMD_CONNECT:
		com.connect(rec, cmd)

	case C.CMD_DISCONNECTED:
		com.disconnect(rec)

	case C.CMD_SPLIT:
		com.teo.split.cmdSplit(rec)

	case C.CMD_RESET:
		com.reset(rec)

	case C.CMD_ECHO:
		com.echo(rec)

	case C.CMD_ECHO_ANSWER:
		com.echoAnswer(rec)

	case C.CMD_L0:
		com.teo.l0.cmdL0(rec)

	case C.CMD_L0_TO:
		com.teo.l0.cmdL0To(rec)

	case C.CMD_L0_AUTH:
		com.teo.l0.auth.cmdL0Auth(rec)

	case C.CMD_L0_CLIENTS:
		com.teo.l0.cmdL0Clients(rec)

	case C.CMD_L0_CLIENTS_N:
		com.teo.l0.cmdL0ClientsNumber(rec)

	case C.CMD_PEERS:
		com.peers(rec)

	case C.CMD_HOST_INFO:
		com.hostInfo(rec)

	case C.CMD_HOST_INFO_ANSWER:
		com.hostInfoAnswer(rec)
		processed = false

	case C.CMD_AUTH:
		com.teo.l0.auth.cmdAuth(rec)

	default:
		com.log(rec.rd, "UNKNOWN command")
		processed = false
	}

	// Process waitFrom commands
	if !processed {
		processed = com.teo.wcom.check(rec) > 0
	}

	// Send (not processed) command to user level
	if !processed {
		teolog.DebugVf(MODULE, "got packet: cmd %d from %s, data len: %d\n",
			rec.rd.Cmd(), rec.rd.From(), len(rec.rd.Data()))
		com.teo.ev.send(EventReceived, rec.rd.Packet())
	}

	return
}

// isJSONRequest return true if request command ask JSON in answer
func (com *command) isJSONRequest(data []byte) (isJSON bool) {
	if l := len(JSON); len(data) >= l &&
		bytes.Equal(data[:l], JSON) {
		//(bytes.Equal(data[:l], JSON) || bytes.Equal(data[:l+2], JSONs)) {
		isJSON = true
	}
	return
}

// log command processed log message
func (com *command) log(rd *C.ksnCorePacketData, descr string) {
	teolog.DebugVfd(1, MODULE, "got cmd: %d, from: %s, data_len: %d (%s)",
		rd.Cmd(), rd.From(), rd.DataLen(), descr)
}

// error command processed with error log message
func (com *command) error(rd *C.ksnCorePacketData, descr string) {
	teolog.Errorfd(1, MODULE, "got cmd: %d, from: %s, data_len: %d (%s)",
		rd.Cmd(), rd.From(), rd.DataLen(), descr)
}

// connect process 'connect' command and answer with 'connect' command
func (com *command) connect(rec *receiveData, cmd byte) {
	if cmd == C.CMD_CONNECT {
		var to string
		if rec.rd != nil && rec.rd.Data() != nil {
			peer, addr, port, err := com.teo.rhost.cmdConnectData(rec)
			if err == nil {
				to = fmt.Sprintf("%s %s:%d", peer, addr, port)
			}
		}
		com.log(rec.rd, "CMD_CONNECT command: "+to)
		com.teo.rhost.cmdConnect(rec)
	} else {
		com.log(rec.rd, "CMD_NONE command")
		//com.teo.sendToTcd(rec.tcd, C.CMD_HOST_INFO, []byte{0})
		//com.teo.sendToTcd(rec.tcd, C.CMD_NONE, []byte{0})
	}
	// \TODO ??? send 'connected' event to user level
}

// disconnect process 'disconnect' command and close trudp channel and delete
// peer from arp table
func (com *command) disconnect(rec *receiveData) {
	com.log(rec.rd, fmt.Sprint("CMD_DISCONNECTED command ", rec.rd.Data(), string(rec.rd.Data())))
	com.teo.arp.delete(rec)
	// \TODO send 'disconnected' event to user level
}

// reset process 'reset' command data: <t byte>
//   t = 0 - soft reset
//   t = 1 - hard reset
func (com *command) reset(rec *receiveData) {
	com.log(rec.rd, "CMD_RESET command")
	if rec.rd.DataLen() > 0 {
		b := rec.rd.Data()[0]
		if b == 1 || b == '1' {
			com.teo.Reconnect()
		}
	}
}

// echo process 'echo' command and answer with 'echo answer' command
func (com *command) echo(rec *receiveData) {
	com.log(rec.rd, "CMD_ECHO command, data: "+
		C.GoString((*C.char)(unsafe.Pointer(&rec.rd.Data()[0]))))
	com.teo.sendAnswer(rec, C.CMD_ECHO_ANSWER, rec.rd.Data())
}

// echo process 'echoAnswer' command
func (com *command) echoAnswer(rec *receiveData) {
	com.log(rec.rd, "CMD_ECHO_ANSWER command, data: "+
		C.GoString((*C.char)(unsafe.Pointer(&rec.rd.Data()[0]))))
}

// hostInfo is the host info json data structure
type hostInfo struct {
	Name        string   `json:"name"`
	Type        []string `json:"type"`
	AppType     []string `json:"appType"`
	Version     string   `json:"version"`
	AppVersion1 string   `json:"app_version"`
	AppVersion2 string   `json:"appVersion"`
}

// hostInfo process 'hostInfo' command and send host info to peer from
func (com *command) hostInfo(rec *receiveData) (err error) {
	var data []byte

	// Select this host in arp table
	peerArp, ok := com.teo.arp.m[com.teo.param.Name]
	if !ok {
		err = errors.New("host " + com.teo.param.Name + " does not exist in arp table")
		com.error(rec.rd, "CMD_HOST_INFO command processed with error: "+err.Error())
		return
	}
	com.log(rec.rd, "CMD_HOST_INFO command")

	// This func convert string Version to byte array
	ver := func(version string) (data []byte) {
		ver := strings.Split(com.teo.Version(), ".")
		for _, vstr := range ver {
			v, _ := strconv.Atoi(vstr)
			data = append(data, byte(v))
		}
		return
	}

	// Create Json or bynary answer depend of input data: JSON - than answer in json
	if com.isJSONRequest(rec.rd.Data()) {
		data, _ = json.Marshal(hostInfo{com.teo.param.Name, peerArp.appType,
			peerArp.appType, com.teo.Version(), peerArp.appVersion, peerArp.appVersion})
		data = append(data, 0) // add trailing zero (cstring)
	} else {
		typeArLen := len(peerArp.appType)
		name := com.teo.param.Name
		data = ver(com.teo.Version())                   // Version
		data = append(data, byte(typeArLen+1))          // Types array length
		data = append(data, append([]byte(name), 0)...) // Name
		for i := 0; i < typeArLen; i++ {                // Types array
			data = append(data, append([]byte(peerArp.appType[i]), 0)...)
		}
	}

	// Send answer with host infor data
	//com.teo.sendToTcd(rec.tcd, C.CMD_HOST_INFO_ANSWER, data)
	com.teo.sendAnswer(rec, C.CMD_HOST_INFO_ANSWER, data)

	return
}

// hostInfoAnswer process 'hostInfoAnswer' command and add host info to the arp table
func (com *command) hostInfoAnswer(rec *receiveData) (err error) {
	data := rec.rd.Data()
	var typeAr []string
	var version string

	// Parse json or binary format depend of data.
	// If first char = '{' and last char = '}' than data is in json
	if l := len(data); l > 3 && data[0] == '{' && data[l-2] == '}' && data[l-1] == 0 {
		var j hostInfo
		json.Unmarshal(data, &j)
		version = j.Version
		typeAr = append([]string{j.Name}, j.Type...)
	} else {
		version = strconv.Itoa(int(data[0])) + "." + strconv.Itoa(int(data[1])) + "." + strconv.Itoa(int(data[2]))
		typeArLen := int(data[3])
		ptr := 4
		for i := 0; i < typeArLen; i++ {
			charPtr := unsafe.Pointer(&data[ptr])
			typeAr = append(typeAr, C.GoString((*C.char)(charPtr)))
			ptr += len(typeAr[i]) + 1
		}
	}

	// Save to arp Table
	peerArp, ok := com.teo.arp.m[rec.rd.From()]
	if !ok {
		err = errors.New("peer " + rec.rd.From() + " does not exist in arp table")
		com.error(rec.rd, "CMD_HOST_INFO_ANSWER command processed with error: "+err.Error())
		return
	}
	com.log(rec.rd, "CMD_HOST_INFO_ANSWER command")
	peerArp.version = version
	peerArp.appType = typeAr[1:]
	com.teo.arp.print()

	return
}

// peers process 'peers' command
func (com *command) peers(rec *receiveData) (err error) {
	com.log(rec.rd, "CMD_PEERS command")

	var data []byte

	// Get type of request: 0 - binary; 1 - JSON
	if com.isJSONRequest(rec.rd.Data()) {
		data, _ = com.teo.arp.json()
	} else {
		data, _ = com.teo.arp.binary()
	}

	// \TODO: create peers answer on binary and json format. Create functions in
	// arp module to generate peers structure
	com.teo.sendAnswer(rec, C.CMD_PEERS_ANSWER, data)
	return
}

// RemoveTrailingZero remove trailing zero in byte slice
func RemoveTrailingZero(data []byte) []byte { com := &command{}; return com.removeTrailingZero(data) }

// DataIsJSON simple check that data is JSON string
func DataIsJSON(data []byte) bool { com := &command{}; return com.dataIsJSON(data) }

// removeTrailingZero remove trailing zero in byte slice
func (com *command) removeTrailingZero(data []byte) []byte {
	if l := len(data); l > 0 && data[l-1] == 0 {
		data = data[:l-1]
	}
	return data
}

// dataIsJSON simple check that data is JSON string
func (com *command) dataIsJSON(data []byte) bool {
	data = com.removeTrailingZero(data)
	return len(data) >= 2 && (data[0] == '{' && data[len(data)-1] == '}' ||
		data[0] == '[' && data[len(data)-1] == ']')
}

// marshalClients convert binary client list data to json,
// cmd: CMD_L0_CLIENTS_ANSWER #80
func (com *command) marshalClients(data []byte) (js []byte) {
	var dataLen C.size_t
	jstr := C.marshalClients(unsafe.Pointer(&data[0]), &dataLen)
	jstrPtr := unsafe.Pointer(jstr)
	js = C.GoBytes(jstrPtr, C.int(dataLen))
	C.free(jstrPtr)
	return
}

// marshalSubscribe convert binary subscribe answer data to json
// cmd: CMD_L_SUBSCRIBE_ANSWER #83
func (com *command) marshalSubscribe(data []byte) (js []byte) {
	var dataLen C.size_t
	jstr := C.marshalSubscribe(unsafe.Pointer(&data[0]), C.size_t(len(data)), &dataLen)
	jstrPtr := unsafe.Pointer(jstr)
	js = C.GoBytes(jstrPtr, C.int(dataLen))
	C.free(jstrPtr)
	return
}

// marshalClientsNum convert binary clients number data to json,
// cmd: CMD_L0_CLIENTS_N_ANSWER #85
func (com *command) marshalClientsNum(data []byte) (js []byte) {
	numClients := binary.LittleEndian.Uint32(data[:unsafe.Sizeof(uint32(0))])
	js = []byte(fmt.Sprintf(`{"numClients":%d}`, numClients))
	return
}
