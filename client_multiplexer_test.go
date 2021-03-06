package quic

import (
	"bytes"
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/internal/wire"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client Multiplexer", func() {
	getPacket := func(connID protocol.ConnectionID) []byte {
		buf := &bytes.Buffer{}
		err := (&wire.Header{
			DestConnectionID: connID,
			PacketNumberLen:  protocol.PacketNumberLen1,
		}).Write(buf, protocol.PerspectiveServer, protocol.VersionWhatever)
		Expect(err).ToNot(HaveOccurred())
		return buf.Bytes()
	}

	It("adds a new packet conn and handles packets", func() {
		conn := newMockPacketConn()
		connID := protocol.ConnectionID{1, 2, 3, 4, 5, 6, 7, 8}
		packetHandler := NewMockQuicSession(mockCtrl)
		handledPacket := make(chan struct{})
		packetHandler.EXPECT().handlePacket(gomock.Any()).Do(func(p *receivedPacket) {
			Expect(p.header.DestConnectionID).To(Equal(connID))
			close(handledPacket)
		})
		packetHandler.EXPECT().GetVersion()
		getClientMultiplexer().AddConn(conn, 8)
		err := getClientMultiplexer().AddHandler(conn, connID, packetHandler)
		Expect(err).ToNot(HaveOccurred())
		conn.dataToRead <- getPacket(connID)
		Eventually(handledPacket).Should(BeClosed())
		// makes the listen go routine return
		packetHandler.EXPECT().Close(gomock.Any()).AnyTimes()
		close(conn.dataToRead)
	})

	It("errors when adding an existing conn with a different connection ID length", func() {
		conn := newMockPacketConn()
		_, err := getClientMultiplexer().AddConn(conn, 5)
		Expect(err).ToNot(HaveOccurred())
		_, err = getClientMultiplexer().AddConn(conn, 6)
		Expect(err).To(MatchError("cannot use 6 byte connection IDs on a connection that is already using 5 byte connction IDs"))
	})

	It("errors when adding a handler for an unknown conn", func() {
		conn := newMockPacketConn()
		err := getClientMultiplexer().AddHandler(conn, protocol.ConnectionID{1, 2, 3, 4}, NewMockQuicSession(mockCtrl))
		Expect(err).ToNot(MatchError("unknown packet conn"))
	})

	It("handles packets for different packet handlers on the same packet conn", func() {
		conn := newMockPacketConn()
		connID1 := protocol.ConnectionID{1, 2, 3, 4, 5, 6, 7, 8}
		connID2 := protocol.ConnectionID{8, 7, 6, 5, 4, 3, 2, 1}
		packetHandler1 := NewMockQuicSession(mockCtrl)
		packetHandler2 := NewMockQuicSession(mockCtrl)
		handledPacket1 := make(chan struct{})
		handledPacket2 := make(chan struct{})
		packetHandler1.EXPECT().handlePacket(gomock.Any()).Do(func(p *receivedPacket) {
			Expect(p.header.DestConnectionID).To(Equal(connID1))
			close(handledPacket1)
		})
		packetHandler1.EXPECT().GetVersion()
		packetHandler2.EXPECT().handlePacket(gomock.Any()).Do(func(p *receivedPacket) {
			Expect(p.header.DestConnectionID).To(Equal(connID2))
			close(handledPacket2)
		})
		packetHandler2.EXPECT().GetVersion()
		getClientMultiplexer().AddConn(conn, connID1.Len())
		Expect(getClientMultiplexer().AddHandler(conn, connID1, packetHandler1)).To(Succeed())
		Expect(getClientMultiplexer().AddHandler(conn, connID2, packetHandler2)).To(Succeed())

		conn.dataToRead <- getPacket(connID1)
		conn.dataToRead <- getPacket(connID2)
		Eventually(handledPacket1).Should(BeClosed())
		Eventually(handledPacket2).Should(BeClosed())

		// makes the listen go routine return
		packetHandler1.EXPECT().Close(gomock.Any()).AnyTimes()
		packetHandler2.EXPECT().Close(gomock.Any()).AnyTimes()
		close(conn.dataToRead)
	})

	It("drops unparseable packets", func() {
		conn := newMockPacketConn()
		connID := protocol.ConnectionID{1, 2, 3, 4, 5, 6, 7}
		conn.dataToRead <- []byte("invalid header")
		packetHandler := NewMockQuicSession(mockCtrl)
		getClientMultiplexer().AddConn(conn, 7)
		Expect(getClientMultiplexer().AddHandler(conn, connID, packetHandler)).To(Succeed())
		time.Sleep(100 * time.Millisecond) // give the listen go routine some time to process the packet
		packetHandler.EXPECT().Close(gomock.Any()).AnyTimes()
		close(conn.dataToRead)
	})

	It("ignores packets arriving late for closed sessions", func() {
		manager := NewMockPacketHandlerManager(mockCtrl)
		origNewPacketHandlerManager := getClientMultiplexer().(*clientMultiplexer).newPacketHandlerManager
		defer func() {
			getClientMultiplexer().(*clientMultiplexer).newPacketHandlerManager = origNewPacketHandlerManager
		}()
		getClientMultiplexer().(*clientMultiplexer).newPacketHandlerManager = func() packetHandlerManager { return manager }

		conn := newMockPacketConn()
		connID := protocol.ConnectionID{1, 2, 3, 4, 5, 6, 7, 8}
		done := make(chan struct{})
		manager.EXPECT().Get(connID).Do(func(protocol.ConnectionID) { close(done) }).Return(nil, true)
		getClientMultiplexer().AddConn(conn, 8)
		conn.dataToRead <- getPacket(connID)
		Eventually(done).Should(BeClosed())
		// makes the listen go routine return
		manager.EXPECT().Close(gomock.Any()).AnyTimes()
		close(conn.dataToRead)
	})

	It("drops packets for unknown receivers", func() {
		conn := newMockPacketConn()
		conn.dataToRead <- getPacket(protocol.ConnectionID{1, 2, 3, 4, 5, 6, 7, 8})
		packetHandler := NewMockQuicSession(mockCtrl)
		getClientMultiplexer().AddConn(conn, 8)
		Expect(getClientMultiplexer().AddHandler(conn, protocol.ConnectionID{8, 7, 6, 5, 4, 3, 2, 1}, packetHandler)).To(Succeed())
		time.Sleep(100 * time.Millisecond) // give the listen go routine some time to process the packet
		// makes the listen go routine return
		packetHandler.EXPECT().Close(gomock.Any()).AnyTimes()
		close(conn.dataToRead)
	})

	It("closes the packet handlers when reading from the conn fails", func() {
		conn := newMockPacketConn()
		testErr := errors.New("test error")
		conn.readErr = testErr
		done := make(chan struct{})
		packetHandler := NewMockQuicSession(mockCtrl)
		packetHandler.EXPECT().Close(testErr).Do(func(error) {
			close(done)
		})
		getClientMultiplexer().AddConn(conn, 8)
		Expect(getClientMultiplexer().AddHandler(conn, protocol.ConnectionID{1, 2, 3, 4, 5, 6, 7, 8}, packetHandler)).To(Succeed())
		Eventually(done).Should(BeClosed())
	})
})
