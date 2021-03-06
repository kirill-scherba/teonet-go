// Copyright 2019 teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package trudp is the Teonet relable udp processing package.
//
package trudp

import (
	"net"
	"strconv"
	"time"

	"github.com/kirill-scherba/teonet-go/teokeys/teokeys"
	"github.com/kirill-scherba/teonet-go/teolog/teolog"
)

// Version is srudp version
const Version = "3.0.0"

// MODULE trudp packet name in logs
var MODULE = teokeys.Color(teokeys.ANSICyan, "(trudp)")

const (
	maxResendAttempt = 50               // (number) max number of resend packet from sendQueue
	maxBufferSize    = 2048             // (bytes) send buffer size in bytes
	pingAfter        = 1000             // (ms) send ping afret in ms
	disconnectAfter  = 3000             // (ms) disconnect afret in ms
	defaultRTT       = 30               // (ms) default retransmit time in ms
	maxRTT           = 500              // (ms) default maximum time in ms
	firstPacketID    = 0                // (number) first packet ID and first expectedID number
	chRWUdpSize      = 1024             // Size of read and write channel used to got/send data from udp
	chWriteSize      = 256              // Size of writer channel used to send data from users level and than send it to remote host
	maxRQueue        = 65536            // Max size of receive queue
	chEventSize      = 2048 + maxRQueue // Size or read channel used to send messages to user level

	// DefaultQueueSize is size of send and receive queue
	DefaultQueueSize = 256 // 96

	helloMsg      = "hello"
	echoMsg       = "ping\x00"
	echoAnswerMsg = "pong\x00"

	// Network time & local host name
	network  = "udp"
	hostName = ""
)

// TRUDP connection strucure
type TRUDP struct {

	// UDP address and functions
	udp *udp

	// Control maps, channels and function holder
	tcdmap      map[string]*ChannelData // channel data map
	chanEvent   chan *EventData         // User level event channel
	allowEvents uint32                  // allow send events \TODO: use flags
	packet      *packetType             // packet functions holder
	ticker      *time.Ticker            // timer ticler
	proc        *process                // process container

	// Logger configuration
	logLevel int  // trudp log level
	logLogF  bool // show time in trudp log

	// Statistic
	startTime time.Time   // TRUDP start running time
	packets   packetsStat // TRUDP packets statistic

	defaultQueueSize int // Default queues size

	// Control Flags
	showStatF bool // Show statistic
}

// trudpStat structure contain trudp statistic variables
type packetsStat struct {
	send          uint32        // Total packets send
	sendLength    uint64        // Total send in bytes
	ack           uint32        // Total ACK reseived
	receive       uint32        // Total packet reseived
	receiveLength uint64        // Total reseived in bytes
	dropped       uint32        // Total packet droped
	repeat        uint32        // Total packet repeated
	sendRT        RealTimeSpeed // Send real time speed
	receiveRT     RealTimeSpeed // Receive real time speed
	repeatRT      RealTimeSpeed // Repiat real time speed
}

// EventData used as structure in sendEvent function
type EventData struct {
	Tcd   *ChannelData
	Event int
	Data  []byte
}

// Enumeration of Trudp events
const (

	/**
	 * Initialize TR-UDP event
	 * @param td Pointer to trudpData
	 */
	EvInitialize = iota

	/**
	 * Destroy TR-UDP event
	 * @param td Pointer to trudpData
	 */
	EvDestroy

	/**
	 * TR-UDP channel disconnected event
	 * @param data NULL
	 * @param data_length 0
	 * @param user_data NULL
	 */
	EvConnected

	/**
	 * TR-UDP channel disconnected event
	 * @param data Last packet received
	 * @param data_length 0
	 * @param user_data NULL
	 */
	EvDisconnected

	/**
	 * Got TR-UDP reset packet
	 * @param data NULL
	 * @param data_length 0
	 * @param user_data NULL
	 */
	EvGotReset

	/**
	 * Send TR-UDP reset packet
	 * @param data Pointer to uint32_t send id or NULL if received id = 0
	 * @param data_length Size of uint32_t or 0
	 * @param user_data NULL
	 */
	EvSendReset

	/**
	 * Got ACK to reset command
	 * @param data NULL
	 * @param data_length 0
	 * @param user_data NULL
	 */
	EvGotAckReset

	/**
	 * Got ACK to ping command
	 * @param data Pointer to ping data (usually it is a string)
	 * @param data_length Length of data
	 * @param user_data NULL
	 */
	EvGotAckPing

	/**
	 * Got PING command
	 * @param data Pointer to ping data (usually it is a string)
	 * @param data_length Length of data
	 * @param user_data NULL
	 */
	EvGotPing

	/**
	 * Got ACK command
	 * @param data Pointer to ACK packet
	 * @param data_length Length of data
	 * @param user_data NULL
	 */
	EvGotAck

	/**
	 * Got DATA
	 * @param data Pointer to data
	 * @param data_length Length of data
	 * @param user_data NULL
	 */
	EvGotData
	EvGotDataNotrudp

	/**
	 * Process received data
	 * @param tcd Pointer to trudpData
	 * @param data Pointer to receive buffer
	 * @param data_length Receive buffer length
	 * @param user_data NULL
	 */
	EvProcessReceived

	/** Process received not TR-UDP data
	 * @param tcd Pointer to trudpData
	 * @param data Pointer to receive buffer
	 * @param data_length Receive buffer length
	 * @param user_data NULL
	 */
	EvProcessReceivedNoTrudp

	/** Process send data
	 * @param data Pointer to send data
	 * @param data_length Length of send
	 * @param user_data NULL
	 */
	//SEND_DATA

	EvResetLocal
)

// Init start trudp connection
func Init(port *int) (trudp *TRUDP) {

	trudp = &TRUDP{
		udp:              &udp{},
		packet:           &packetType{},
		startTime:        time.Now(),
		tcdmap:           make(map[string]*ChannelData),
		chanEvent:        make(chan *EventData, chEventSize),
		defaultQueueSize: DefaultQueueSize,
	}
	trudp.packet.trudp = trudp

	// Connect to UDP and start UDP workers
	trudp.udp.listen(port)
	trudp.proc = new(process).init(trudp)

	localAddr := trudp.udp.localAddr()
	teolog.Log(teolog.CONNECT, MODULE, "start listenning at", localAddr)
	go trudp.sendEvent(nil, EvInitialize, []byte(localAddr))

	return
}

// sendEventAvailable return true if send event available
func (trudp *TRUDP) sendEventAvailable() bool {
	//return len(trudp.chanEvent) < (chEventSize - maxRQueue - 16)
	return len(trudp.chanEvent) < chEventSize-16
}

// sendEvent Send event to user level (to event callback or channel)
func (trudp *TRUDP) sendEvent(tcd *ChannelData, event int, data []byte) {
	trudp.chanEvent <- &EventData{tcd, event, data}
}

// Connect to remote host by UDP
func (trudp *TRUDP) Connect(rhost string, rport int) {

	service := rhost + ":" + strconv.Itoa(rport)
	rUDPAddr, err := trudp.udp.resolveAddr(network, service)
	if err != nil {
		panic(err)
	}
	teolog.Log(teolog.CONNECT, MODULE, "connecting to host", rUDPAddr)

	// Send hello to remote host
	trudp.udp.writeTo([]byte(helloMsg), rUDPAddr)

	// Keep alive: send Ping
	go func() {
		for {
			time.Sleep(pingAfter * time.Millisecond)
			dt, _ := time.Now().MarshalBinary()
			trudp.udp.writeTo(append([]byte(echoMsg), dt...), rUDPAddr)
		}
	}()
}

// AllowEvents set allow events flags
func (trudp *TRUDP) AllowEvents(events uint32) {
	trudp.allowEvents = events
}

// Run waits some data received from UDP port and procces it
func (trudp *TRUDP) Run() {
	for {
		buffer := make([]byte, maxBufferSize)

		nRead, addr, err := trudp.udp.readFrom(buffer)
		if err != nil {
			teolog.Log(teolog.CONNECT, MODULE, "stop listenning at", trudp.udp.localAddr())
			close(trudp.proc.chanReader)
			trudp.proc.destroy()
			trudp.proc.wg.Wait()
			teolog.Log(teolog.CONNECT, MODULE, "stopped")
			break
		}

		switch {
		// Empty packet
		case nRead == 0:
			teolog.DebugV(MODULE, "empty paket received from:", addr)

		// Check trudp packet
		case trudp.packet.check(buffer[:nRead]):
			packet := &packetType{trudp: trudp, data: buffer[:nRead]}
			trudp.proc.chanReader <- &readerType{addr, packet}

		// Process connect message
		// (this is non-trudp test command, it may be deprecated)
		case nRead == len(helloMsg) &&
			string(buffer[:len(helloMsg)]) == helloMsg:
			teolog.Log(teolog.DEBUG, MODULE, "got", nRead,
				"bytes 'connect' message from:", addr, "data: ", buffer[:nRead],
				string(buffer[:nRead]))

		// Process echo message Ping (send to Pong)
		// (this is non-trudp test command, it may be deprecated)
		case nRead > len(echoMsg) && string(buffer[:len(echoMsg)]) == echoMsg:
			teolog.Log(teolog.DEBUG, MODULE, "got", nRead,
				"byte 'ping' command from:", addr, buffer[:nRead])
			trudp.udp.writeTo(append([]byte(echoAnswerMsg),
				buffer[len(echoMsg):nRead]...), addr)

		// Process echo answer message Pong (answer to Ping)
		// (this is non-trudp test command, it may be deprecated)
		case nRead > len(echoAnswerMsg) &&
			string(buffer[:len(echoAnswerMsg)]) == echoAnswerMsg:
			var ts time.Time
			ts.UnmarshalBinary(buffer[len(echoAnswerMsg):nRead])
			teolog.Log(teolog.DEBUG, MODULE, "got", nRead,
				"byte 'pong' command from:", addr, "trip time:",
				time.Since(ts), buffer[:nRead])

		// Not trudp packet received (it may be teonet not-trudp commands)
		default:
			teolog.DebugVf(MODULE,
				"got (---==Not TRUDP==---) %d bytes, from: %s\n", nRead, addr)
			// Process teonet notTrudp messages if trudp channel exists, or
			// ignore this message if channel does not exsists.
			go trudp.kernel(func() {
				tcd, _, ok := trudp.newChannelData(addr, 0, false, false)
				if !ok {
					return
				}
				tcd.trudp.sendEvent(tcd, EvGotDataNotrudp, buffer[:nRead])
			})
		}
	}
}

// Running return true if TRUDP is running now
func (trudp *TRUDP) Running() bool {
	return !trudp.proc.stopRunningF
}

// closeChannels Close all trudp channels
func (trudp *TRUDP) closeChannels() {
	for key, tcd := range trudp.tcdmap {
		tcd.destroy(teolog.CONNECT, "close "+key)
	}
}

// Close closes trudp connection and channelRead
func (trudp *TRUDP) Close() {
	if trudp.udp.conn != nil {
		trudp.udp.conn.Close()
	}
}

// kernel run function in trudp kernel (main process)
func (trudp *TRUDP) kernel(f func()) {
	// \TODO may be use 'if trudp.Running()' here
	if !trudp.proc.chanKernelF {
		trudp.proc.chanKernel <- f
	}
}

// ChanEvent return channel to read trudp events
func (trudp *TRUDP) ChanEvent() <-chan *EventData {
	trudp.proc.once.Do(func() {
		trudp.proc.wg.Add(1)
	})
	return trudp.chanEvent
}

// ChanEventClosed signalling that event channel reader routine sucessfully closed
func (trudp *TRUDP) ChanEventClosed() {
	teolog.Log(teolog.DEBUG, MODULE, "ChanEventClosed")
	trudp.proc.wg.Done()
}

// SetShowStatistic set showStatF to show trudp statistic window
func (trudp *TRUDP) SetShowStatistic(showStatF bool) {
	trudp.showStatF = showStatF
}

// ShowStatistic get showStatF
func (trudp *TRUDP) ShowStatistic() bool {
	return trudp.showStatF
}

// SetDefaultQueueSize set maximum send and receive queues size
func (trudp *TRUDP) SetDefaultQueueSize(defaultQueueSize int) {
	trudp.defaultQueueSize = defaultQueueSize
}

// GetAddr return IP and Port of local address
func (trudp *TRUDP) GetAddr() (ip string, port int) {
	ip = string(trudp.udp.conn.LocalAddr().(*net.UDPAddr).IP)
	port = trudp.udp.conn.LocalAddr().(*net.UDPAddr).Port
	return
}
