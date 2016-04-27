package congestion

import (
	"time"

	"github.com/lucas-clemente/quic-go/protocol"
)

type SendAlgorithm interface {
	TimeUntilSend(now time.Time, bytesInFlight uint64) time.Duration
	OnPacketSent(sentTime time.Time, bytesInFlight uint64, packetNumber protocol.PacketNumber, bytes uint64, isRetransmittable bool) bool
	GetCongestionWindow() uint64
}