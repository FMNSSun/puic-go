package quic

import "net"

import "github.com/mami-project/plus-lib"
import "github.com/lucas-clemente/quic-go/internal/wire"

var UsePLUS bool = true

type plusconn struct {
	p *PLUS.Connection
	m *PLUS.ConnectionManager
	s *session
}

func (c *plusconn) SetSession(s *session) {
	c.s = s
	c.p.SetFeedbackChannel(c)
}

func (c *plusconn) SendFeedback(fb []byte) error {
	c.s.queuePLUSFeedback(fb)
	return nil
}

func (c *plusconn) Write(p []byte) error {
	_, err := c.p.Write(p)
	return err
}

func (c *plusconn) Read(p []byte) (int, net.Addr, error) {
	return c.p.ReadAndAddr(p)
}

func (c *plusconn) SetCurrentRemoteAddr(addr net.Addr) {
	c.p.SetRemoteAddr(addr)
}

func (c *plusconn) LocalAddr() net.Addr {
	return c.p.LocalAddr()
}

func (c *plusconn) RemoteAddr() net.Addr {
	return c.p.RemoteAddr()
}

func (c *plusconn) Close() error {
	return c.p.Close()
}

func (s *session) queuePLUSFeedback(data []byte) {
	s.queueControlFrame(&wire.PLUSFeedbackFrame{
		Data: data,
	})
}
