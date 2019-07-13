package trudp

import (
	"net"
	"time"
)

// This module process all trudp internal events:
// - read (received from udp),
// - write (received from user level, need write to udp)
// - keep alive timer
// - resend packet from send queue timer

// process data structure
type process struct {
	trudp       *TRUDP           // link to trudp
	chanWrite   chan writeType   // channel to write (used to send data from user level)
	chanRead    chan readType    // channel to read (used to process packets received from udp)
	timerKeep   *time.Ticker     // keep alive timer
	timerResend <-chan time.Time // resend packet from send queue timer
}

// read channel data structure
type readType struct {
	addr   *net.UDPAddr
	packet *packetType
}

// read channel data structure
type writeType struct {
	tcd  *channelData
	data []byte
}

// init
func (proc *process) init() *process {

	// Set time variables
	resendTime := defaultRTT * time.Millisecond
	pingTime := pingInterval * time.Millisecond
	//disconnectTime := disconnectAfter * time.Millisecond

	// Init channels and timers
	proc.chanRead = make(chan readType)
	proc.chanWrite = make(chan writeType)
	//
	proc.timerResend = time.After(resendTime)
	proc.timerKeep = time.NewTicker(pingTime)

	// Do it on return
	defer func() { proc.timerKeep.Stop() }()

	// Module worker
	go func() {
		for {
			select {

			// Process read packet (received from udp)
			case readPac := <-proc.chanRead:
				readPac.packet.process(readPac.addr)

			// Process write packet (received from user level, need write to udp)
			case <-proc.chanWrite:
				//writePac.tcd.trudp.packet.dataCreateNew(writePac.tcd.getID(), writePac.tcd.ch, writePac.data).writeToUnsafe(writePac.tcd)

			case <-proc.timerKeep.C:

			case <-proc.timerResend:
				resendTime = 0
				for _, tcd := range proc.trudp.tcdmap {
					rt := tcd.sendQueueResendProcess()
					if resendTime == 0 || rt < resendTime {
						resendTime = rt
					}
				}
				proc.timerResend = time.After(resendTime) // Set new timer value
			}
		}
	}()

	return proc
}

// destroy
func (proc *process) destroy() {

}
