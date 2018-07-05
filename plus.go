package quic

import "net"

import "github.com/mami-project/plus-lib"

var UsePLUS bool = true

type plusconn struct {
	p *PLUS.Connection
	m *PLUS.ConnectionManager
	queueFunc func([]byte)
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
