package trudp

import (
	"container/list"
	"errors"
	"fmt"
	"time"

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
)

type sendQueueData struct {
	packet        *packetType // packet
	sendTime      time.Time   // time when packet was send
	arrivalTime   time.Time   // time when packet need resend
	resendAttempt int         // number of resend was done
}

// sendQueueResendProcess resend packet from send queue if it does not got
// ACK during selected time. Destroy channel if too much resends happens =
// maxResendAttempt constant
// \TODO check this resend and calculate new resend time algorithm
func (tcd *ChannelData) sendQueueResendProcess() (rtt time.Duration) {
	rtt = (defaultRTT + time.Duration(tcd.stat.triptimeMiddle)) * time.Millisecond
	now := time.Now()
	for e := tcd.sendQueue.Front(); e != nil; e = e.Next() {
		sqd := e.Value.(*sendQueueData)
		// Do while packets ready to resend
		if !now.After(sqd.arrivalTime) {
			tcd.stat.repeat(false)
			break
		}
		// Destroy this trudp channel if resendAttemp more than maxResendAttemp
		if sqd.resendAttempt >= maxResendAttempt {
			tcd.destroy(teolog.DEBUGv, fmt.Sprint("destroy channel ",
				tcd.GetKey(), ": too much resends happens: ",
				sqd.resendAttempt))
			break
		}
		// Resend packet, save resend to statistic and show message
		sqd.packet.updateTimestamp().writeTo(tcd)
		tcd.stat.repeat(true)
		teolog.Log(teolog.DEBUGvv, MODULE, "resend sendQueue packet ",
			"id:", sqd.packet.ID(),
			"attempt:", sqd.resendAttempt)
	}
	// Next time to run sendQueueResendProcess
	if tcd.sendQueue.Len() > 0 {
		rtt = tcd.sendQueue.Front().Value.(*sendQueueData).arrivalTime.Sub(now)
	}
	return
}

// sendQueueAdd add or update send queue packet
func (tcd *ChannelData) sendQueueAdd(packet *packetType) {
	now := time.Now()
	var triptimeMiddle time.Duration
	if tcd.stat.triptimeMiddle > maxRTT {
		triptimeMiddle = maxRTT
	} else {
		triptimeMiddle = time.Duration(tcd.stat.triptimeMiddle)
	}
	var rttTime time.Duration = defaultRTT + triptimeMiddle
	arrivalTime := now.Add(rttTime * time.Millisecond)

	_, sqd, _, err := tcd.sendQueueFind(packet)
	if err != nil {
		tcd.sendQueue.PushBack(&sendQueueData{
			packet:      packet,
			sendTime:    now,
			arrivalTime: arrivalTime,
		})
		teolog.Log(teolog.DEBUGvv, MODULE, "add to send queue, id:",
			packet.ID())
	} else {
		sqd.arrivalTime = arrivalTime
		sqd.resendAttempt++
		teolog.Log(teolog.DEBUGvv, MODULE, "update in send queue, id",
			packet.ID())
	}
}

// sendQueueFind find packet in sendQueue
func (tcd *ChannelData) sendQueueFind(packet *packetType) (e *list.Element,
	sqd *sendQueueData, id uint32, err error) {
	id = packet.ID()
	for e = tcd.sendQueue.Front(); e != nil; e = e.Next() {
		sqd = e.Value.(*sendQueueData)
		if sqd.packet.ID() == id {
			return
		}
	}
	err = errors.New(fmt.Sprint("not found, packet id: ", id))
	return
}

// sendQueueRemove remove packet from send queue
func (tcd *ChannelData) sendQueueRemove(packet *packetType) {
	e, sqd, id, err := tcd.sendQueueFind(packet)
	if err == nil {
		sqd.packet.destroy()
		tcd.sendQueue.Remove(e)
		teolog.Log(teolog.DEBUGvv, MODULE, "remove from send queue, id:", id)
	}
}

// sendQueueCalculateLength calculate send queue length
func (tcd *ChannelData) sendQueueCalculateLength() {
	// Calculate new send queue length if send packets speed more than 30 pac/sec
	if tcd.stat.packets.sendRT.SpeedPacSec > 30 {
		//currentLen := tcd.sendQueue.Len()
		lessMaxSize := tcd.maxQueueSize < 2048 //1024
		queueIsFull := tcd.sendQueue.Len() >= tcd.maxQueueSize
		moreDefaultSize := tcd.maxQueueSize > 4 //tcd.trudp.defaultQueueSize
		//  if queue capacity less max capacity size
		if lessMaxSize {
			// if repeat speed is nil (0 repeat packets during second) and
			// queue is full
			if tcd.stat.packets.repeatRT.SpeedPacSec == 0 && queueIsFull {
				tcd.maxQueueSize += 4
			}
		}
		// if queue capacity more default(minimal) capacity size
		if moreDefaultSize {
			// if repeat speed more than 20 packets per second or
			// if repeat speed more than 10 packets per second and queue is full
			if tcd.stat.packets.repeatRT.SpeedPacSec > 20 ||
				tcd.stat.packets.repeatRT.SpeedPacSec > 10 && queueIsFull {
				tcd.maxQueueSize -= 4
			}
		}
	}
}

// sendQueueReset resets (clear) send queue
func (tcd *ChannelData) sendQueueReset() {
	for e := tcd.sendQueue.Front(); e != nil; e = e.Next() {
		e.Value.(*sendQueueData).packet.destroy()
	}
	tcd.sendQueue.Init()
}
