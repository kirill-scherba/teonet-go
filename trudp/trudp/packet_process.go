package trudp

import (
	"fmt"
	"net"

	"github.com/kirill-scherba/teonet-go/teokeys/teokeys"
	"github.com/kirill-scherba/teonet-go/teolog/teolog"
)

// Packet type
const (
	DATA     = iota //(0x0)
	ACK             //(0x1)
	RESET           //(0x2)
	ACKReset        //(0x3)
	PING            //(0x4)
	ACKPing         //(0x5)
)

// process received packet
func (pac *packetType) process(addr *net.UDPAddr) (processed bool) {
	processed = false

	ch := pac.Channel()
	tcd, key, ok := pac.trudp.newChannelData(addr, ch, pac.Type() == DATA, true)
	if !ok {
		return
	}

	tcd.stat.setLastTimeReceived()

	packetType := pac.Type()
	switch packetType {

	// DATA packet received
	case DATA:

		// \TODO: drop this packet if EQ len >= MaxValue
		// if len(pac.trudp.chanEvent) > 16 {
		// 	break
		// }

		// Show Log
		teolog.DebugVf(MODULE, "got DATA packet id: %d, channel: %s, "+
			"expected id: %d, data_len: %d",
			pac.ID(), key, tcd.expectedID, len(pac.data),
		)

		// Create ACK packet and send it back to sender
		pac.newAck().writeTo(tcd)
		tcd.stat.received(len(pac.data))

		// Process received queue
		pac.packetDataProcess(tcd)

	// ACK-to-data packet received
	case ACK:

		id := pac.ID()

		// Show Log
		teolog.DebugVf(MODULE, "got ACK packet id: %d, channel: %s, "+
			"triptime: %.3f ms\n",
			id, key, tcd.stat.triptime,
		)

		// Set trip time to ChannelData
		tcd.stat.setTriptime(pac.Triptime())
		tcd.stat.ackReceived()

		// Remove packet from send queue
		tcd.sendQueue.Remove(id)
		tcd.trudp.proc.writeFromQueue(tcd)

	// RESET packet received
	case RESET:

		teolog.DebugV(MODULE, "got RESET packet, channel:", key)
		pac.newAckToReset().writeTo(tcd)
		tcd.reset()

	// ACK-to-reset packet received
	case ACKReset:

		teolog.DebugV(MODULE, "got ACK_RESET packet, channel:", key)
		tcd.reset()

	// PING packet received
	case PING:

		// Show Log
		teolog.DebugVf(MODULE, "got PING packet id: %d, channel: %s, data: %s\n",
			pac.ID(), key, string(pac.Data()),
		)
		// Create ACK to ping packet and send it back to sender
		pac.newAckToPing().writeTo(tcd)

	// ACK-to-PING packet received
	case ACKPing:

		// Show Log
		teolog.DebugVf(MODULE, "got ACK_PING packet id: %d, channel: %s, "+
			"triptime: %.3f ms\n",
			pac.ID(), key, tcd.stat.triptime,
		)

		// Set trip time to ChannelData
		triptime := pac.Triptime()
		tcd.stat.setTriptime(triptime)

		// Send event to user level
		if tcd.trudp.allowEvents > 0 { // \TODO use GOT_ACK_PING to check allow this event
			tcd.trudp.sendEvent(tcd, EvGotAckPing, nil) // []byte(fmt.Sprintf("%.3f", triptime)))
		}

	// UNKNOWN packet received
	default:
		teolog.DebugV(MODULE, "UNKNOWN packet received, channel:", key,
			", type:", packetType,
		)
	}

	return
}

const packetIDlimit = 0x100000000

// modSubU module of subtraction
func (pac *packetType) modSubU(arga, argb uint32, mod uint64) int64 {
	sub := (uint64(arga) % mod) + mod - (uint64(argb) % mod)
	return int64(sub % mod)
}

// packetDistance check received packet dispance and return integer value
// lesse than zero than 'id < expectedID' or return integer value more than
// zero than 'id > tcd.expectedID'
func (pac *packetType) packetDistance(expectedID uint32, id uint32) int {
	diff := pac.modSubU(id, expectedID, packetIDlimit)
	if diff < packetIDlimit/2 {
		return int(diff)
	}
	return int(diff - packetIDlimit)
}

// packetDataProcess process received data packet, check receivedQueue and
// send received data and events to user level
func (pac *packetType) packetDataProcess(tcd *ChannelData) {
	id := pac.ID()
	packetDistance := pac.packetDistance(tcd.expectedID, id)
	switch {

	// Valid data packet
	case packetDistance == 0: // id == tcd.expectedID:
		tcd.incID(&tcd.expectedID)
		teolog.DebugV(MODULE, teokeys.Color(teokeys.ANSILightGreen,
			fmt.Sprintf("received valid packet id: %d, channel: %s",
				int(id), tcd.GetKey())))
		// Send received packet data to user level
		tcd.trudp.sendEvent(tcd, EvGotData, pac.Data())
		// Check valid packets in received queue and send it data to user level
		tcd.receiveQueueProcess(func(data []byte) {
			tcd.trudp.sendEvent(tcd, EvGotData, data)
		})

	// Invalid packet (with id = 0)
	case id == firstPacketID:
		teolog.DebugV(MODULE, teokeys.Color(teokeys.ANSILightRed,
			fmt.Sprintf("received invalid packet id: %d (expected id: %d), channel: %s, "+
				"reset locally", id, tcd.expectedID, tcd.GetKey())))
		tcd.reset()                // Reset local
		pac.packetDataProcess(tcd) // Process packet with id 0

	// Invalid packet (when expectedID = 0)
	case tcd.expectedID == firstPacketID:
		teolog.DebugV(MODULE, teokeys.Color(teokeys.ANSILightRed,
			fmt.Sprintf("received invalid packet id: %d (expected id: %d), channel: %s, "+
				"send reset to remote host", id, tcd.expectedID, tcd.GetKey())))
		pac.newReset().writeTo(tcd) // Send reset
		// Send event "RESET was sent" to user level
		tcd.trudp.sendEvent(tcd, EvSendReset, nil)

	// Already processed packet (id < expectedID)
	case packetDistance < 0: //  id < tcd.expectedID:
		teolog.DebugV(MODULE, teokeys.Color(teokeys.ANSILightBlue,
			fmt.Sprintf("skip received packet id: %d, channel: %s, "+
				"already processed", id, tcd.GetKey())))
		// Set statistic REJECTED (already received) packet
		tcd.stat.dropped()

	// Packet with id more than expectedID placed to receive queue and wait
	// previouse packets
	case packetDistance > 0: // id > tcd.expectedID:
		_, ok := tcd.receiveQueue.Find(id)
		if !ok {
			teolog.DebugV(MODULE, teokeys.Color(teokeys.ANSIYellow,
				fmt.Sprintf("put packet id: %d, channel: %s to received queue, "+
					"wait previouse packets", id, tcd.GetKey())))
			tcd.receiveQueue.Add(pac)
			// <<<< Added to fix overload receve queueu
			tcd.receiveQueueProcess(func(data []byte) {
				tcd.trudp.sendEvent(tcd, EvGotData, data)
			})
			// <<<<
		} else {
			teolog.DebugV(MODULE, teokeys.Color(teokeys.ANSILightBlue,
				fmt.Sprintf("skip received packet id: %d, channel: %s, "+
					"already in receive queue", id, tcd.GetKey())))
			// Set statistic REJECTED (already received) packet
			tcd.stat.dropped()
		}
	}
}
