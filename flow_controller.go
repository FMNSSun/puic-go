package quic

import (
	"sync"

	"github.com/lucas-clemente/quic-go/handshake"
	"github.com/lucas-clemente/quic-go/protocol"
)

type flowController struct {
	streamID protocol.StreamID

	bytesSent                protocol.ByteCount
	sendFlowControlWindow    protocol.ByteCount
	lastBlockedSentForOffset protocol.ByteCount

	bytesRead                         protocol.ByteCount
	highestReceived                   protocol.ByteCount
	receiveWindowUpdateThreshold      protocol.ByteCount
	receiveFlowControlWindow          protocol.ByteCount
	receiveFlowControlWindowIncrement protocol.ByteCount

	mutex sync.RWMutex
}

func newFlowController(streamID protocol.StreamID, connectionParametersManager *handshake.ConnectionParametersManager) *flowController {
	fc := flowController{
		receiveWindowUpdateThreshold:      protocol.WindowUpdateThreshold,
		receiveFlowControlWindowIncrement: protocol.ReceiveStreamFlowControlWindowIncrement,
	}

	if streamID == 0 {
		fc.sendFlowControlWindow = connectionParametersManager.GetSendConnectionFlowControlWindow()
		fc.receiveFlowControlWindow = connectionParametersManager.GetReceiveConnectionFlowControlWindow()
	} else {
		fc.sendFlowControlWindow = connectionParametersManager.GetSendStreamFlowControlWindow()
		fc.receiveFlowControlWindow = connectionParametersManager.GetReceiveStreamFlowControlWindow()
	}

	return &fc
}

func (c *flowController) AddBytesSent(n protocol.ByteCount) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.bytesSent += n
}

// UpdateSendWindow should be called after receiving a WindowUpdateFrame
// it returns true if the window was actually updated
func (c *flowController) UpdateSendWindow(newOffset protocol.ByteCount) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if newOffset > c.sendFlowControlWindow {
		c.sendFlowControlWindow = newOffset
		return true
	}
	return false
}

func (c *flowController) SendWindowSize() protocol.ByteCount {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.bytesSent > c.sendFlowControlWindow { // should never happen, but make sure we don't do an underflow here
		return 0
	}
	return c.sendFlowControlWindow - c.bytesSent
}

// UpdateHighestReceived updates the highestReceived value, if the byteOffset is higher
// Should **only** be used for the stream-level FlowController
func (c *flowController) UpdateHighestReceived(byteOffset protocol.ByteCount) protocol.ByteCount {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if byteOffset > c.highestReceived {
		increment := byteOffset - c.highestReceived
		c.highestReceived = byteOffset
		return increment
	}
	return 0
}

// IncrementHighestReceived adds an increment to the highestReceived value
// Should **only** be used for the connection-level FlowController
func (c *flowController) IncrementHighestReceived(increment protocol.ByteCount) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.highestReceived += increment
}

func (c *flowController) AddBytesRead(n protocol.ByteCount) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.bytesRead += n
}

// MaybeTriggerBlocked determines if it is necessary to send a Blocked for this stream
// it makes sure that only one Blocked is sent for each offset
func (c *flowController) MaybeTriggerBlocked() bool {
	if c.SendWindowSize() != 0 {
		return false
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.lastBlockedSentForOffset == c.sendFlowControlWindow {
		return false
	}

	c.lastBlockedSentForOffset = c.sendFlowControlWindow
	return true
}

// MaybeTriggerWindowUpdate determines if it is necessary to send a WindowUpdate
// if so, it returns true and the offset of the window
func (c *flowController) MaybeTriggerWindowUpdate() (bool, protocol.ByteCount) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	diff := c.receiveFlowControlWindow - c.bytesRead
	if diff < c.receiveWindowUpdateThreshold {
		c.receiveFlowControlWindow += c.receiveFlowControlWindowIncrement
		return true, c.bytesRead + c.receiveFlowControlWindowIncrement
	}
	return false, 0
}

func (c *flowController) CheckFlowControlViolation() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.highestReceived > c.receiveFlowControlWindow {
		return true
	}
	return false
}