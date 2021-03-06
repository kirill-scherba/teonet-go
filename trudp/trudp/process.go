package trudp

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
)

// This module process all trudp internal events:
// - read (received from udp),
// - write (received from user level, need write to udp)
// - keep alive timer
// - resend packet from send queue timer
// - show statistic timer

// process data structure
type process struct {
	trudp       *TRUDP           // link to trudp
	chanReader  chan *readerType // channel to read (used to process packets received from udp)
	chanWrite   chan *writeType  // channel to write (used to send data from user level)
	chanWriter  chan *writerType // channel to write (used to write data to udp)
	chanKernel  chan func()      // channel to execute function on kernel level
	chanKernelF bool             // channels closed flag
	timerResend <-chan time.Time // resend packet from send queue timer

	stopRunningF bool           // Stop running flag
	once         sync.Once      // Once to sync trudp event channel stop
	wg           sync.WaitGroup // Wait group
}

// read channel data structure
type readerType struct {
	addr   *net.UDPAddr
	packet *packetType
}

// read channel data structure
type writeType struct {
	tcd        *ChannelData
	data       []byte
	chanAnswer chan bool
}

type writerType struct {
	packet *packetType
	addr   *net.UDPAddr
}

const disconnectTime = disconnectAfter * time.Millisecond
const sleepTime = pingAfter * time.Millisecond

// init
func (proc *process) init(trudp *TRUDP) *process {

	proc.trudp = trudp

	// Set time variables
	resendTime := defaultRTT * time.Millisecond

	// Init channels and timers
	proc.chanKernel = make(chan func())                   // run in kernel channel
	proc.chanReader = make(chan *readerType, chRWUdpSize) // read from udp channel
	proc.chanWriter = make(chan *writerType, chRWUdpSize) // write to udp channel
	proc.chanWrite = make(chan *writeType, chWriteSize)   // write from user level
	//
	proc.timerResend = time.After(resendTime)

	var ebzdik1 = 0

	// Module worker
	go func() {

		teolog.Log(teolog.CONNECT, MODULE, "process worker started")
		proc.wg.Add(1)

		// Do it on return
		defer func() {
			close(proc.chanWriter)
			teolog.Log(teolog.CONNECT, MODULE, "process worker stopped")

			// Close trudp channels, send DESTROY event and close event channel
			trudp.closeChannels()
			trudp.sendEvent(nil, EvDestroy, []byte(trudp.udp.localAddr()))
			close(trudp.chanEvent)
			proc.chanKernelF = true
			close(proc.chanKernel)

			proc.wg.Done()
		}()

		chanWriteClosedF := false

		for i := 0; ; {
			select {

			// Process read packet (received from udp)
			case readPac, ok := <-proc.chanReader:
				if !ok {
					if !chanWriteClosedF {
						chanWriteClosedF = true
						close(trudp.proc.chanWrite)
					}
					break
				}
				// Process packets if chanEvent is available receive it to avoid deadlock
				// \TODO: May be drop only data packets but process asks and packets
				// which we wait to free receive queue
				if trudp.sendEventAvailable() {
					readPac.packet.process(readPac.addr)
					ebzdik1 = 0
				} else {
					teolog.Error(MODULE, "ebzdik-1 chanEvent len: "+
						strconv.Itoa(len(trudp.chanEvent))+" <===- ", ebzdik1)
					ebzdik1++
				}

			// Process write packet (received from user level, need write to udp)
			case writePac, ok := <-proc.chanWrite:
				if !ok {
					return
				}
				proc.writeTo(writePac)

			case f, ok := <-proc.chanKernel:
				if !ok {
					return
				}
				f()

			// Process send queue (resend packets from send queue), check Keep
			// alive and show statistic (check after 30 ms)
			case <-proc.timerResend:
				// Loop trudp channels map and check Resend send queue and/or
				// send keep alive signal (ping)
				for _, tcd := range proc.trudp.tcdmap {
					// Resend
					tcd.sendQueueResendProcess()
					// Keep alive (every 33*30ms = 990ms)
					if i%33 == 0 {
						tcd.keepAlive()
					}
					// Calculate sendQueue size (every 3*30ms = 90ms)
					if i%3 == 0 {
						tcd.sendQueueCalculateLength()
					}
				}
				// Show statistic window (every 3*30ms = 90ms)
				if i%3 == 0 {
					proc.showStatistic()
				}
				proc.timerResend = time.After(resendTime) // Set new timer value
				i++
			}
		}
	}()

	// Write worker
	go func() {
		proc.wg.Add(1)
		teolog.Log(teolog.CONNECT, MODULE, "writer worker started")
		defer func() {
			teolog.Log(teolog.CONNECT, MODULE, "writer worker stopped")
			proc.wg.Done()
		}()
		for w := range proc.chanWriter {
			proc.trudp.udp.writeTo(w.packet.data, w.addr)
			if !w.packet.sendQueueF {
				w.packet.destroy()
			}
		}
	}()

	return proc
}

// writeTo write packet to trudp channel or write packet to write queue
func (proc *process) writeTo(writePac *writeType) {
	tcd := writePac.tcd
	if len(tcd.writeQueue) == 0 && tcd.canWrite() {
		proc.writeToDirect(writePac)
	} else {
		proc.writeToQueue(tcd, writePac)
	}
}

// writeToDirect write packet to trudp channel and send true to Answer channel
func (proc *process) writeToDirect(writePac *writeType) {
	tcd := writePac.tcd
	proc.trudp.packet.newData(tcd.ID(), tcd.ch, writePac.data).writeTo(tcd)
	writePac.chanAnswer <- true
}

// writeToQueue add write packet to write queue
func (proc *process) writeToQueue(tcd *ChannelData, writePac *writeType) {
	tcd.writeQueue = append(tcd.writeQueue, writePac)
}

// writeFromQueue get packet from writeQueue and send it to trudp channel
func (proc *process) writeFromQueue(tcd *ChannelData) {
	first := true
	isfirst := func() bool {
		if first {
			first = false
			return true
		}
		return false
	}
	for len(tcd.writeQueue) > 0 && (isfirst() || tcd.canWrite()) {
		writePac := tcd.writeQueue[0]
		tcd.writeQueue = tcd.writeQueue[1:]
		proc.writeToDirect(writePac)
	}
}

func (proc *process) writeQueueReset(tcd *ChannelData) {
	for _, writePac := range tcd.writeQueue {
		writePac.chanAnswer <- false
	}
	tcd.writeQueue = tcd.writeQueue[:0]
}

func (proc *process) showStatistic() {
	trudp := proc.trudp
	if !trudp.showStatF {
		return
	}
	idx := 0
	t := time.Now()
	var str string

	// Read trudp channels map keys to slice and sort it
	keys := make([]string, len(trudp.tcdmap))
	for key := range trudp.tcdmap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Get trudp channels statistic string by sorted keys
	for _, key := range keys {
		tcd, ok := trudp.tcdmap[key]
		if ok {
			str += tcd.stat.statBody(tcd, idx, 0)
			idx++
		}
	}

	// Get fotter and print statistic string
	tcs := &channelStat{trudp: trudp} // Empty Methods holder
	str = tcs.statHeader(time.Since(trudp.startTime), time.Since(t)) + str +
		tcs.statFooter(idx)
	fmt.Print(str)
}

// destroy
func (proc *process) destroy() {
	proc.stopRunningF = true
}
