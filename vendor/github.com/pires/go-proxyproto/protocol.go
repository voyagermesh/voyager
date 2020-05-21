package proxyproto

import (
	"bufio"
	"net"
	"sync"
	"time"
)

// Listener is used to wrap an underlying listener,
// whose connections may be using the HAProxy Proxy Protocol.
// If the connection is using the protocol, the RemoteAddr() will return
// the correct client address.
type Listener struct {
	Listener       net.Listener
	Policy         PolicyFunc
	ValidateHeader Validator
}

// Conn is used to wrap and underlying connection which
// may be speaking the Proxy Protocol. If it is, the RemoteAddr() will
// return the address of the client instead of the proxy address.
type Conn struct {
	bufReader         *bufio.Reader
	conn              net.Conn
	header            *Header
	once              sync.Once
	ProxyHeaderPolicy Policy
	Validate          Validator
	readErr           error
}

// Validator receives a header and decides whether it is a valid one
// In case the header is not deemed valid it should return an error.
type Validator func(*Header) error

// ValidateHeader adds given validator for proxy headers to a connection when passed as option to NewConn()
func ValidateHeader(v Validator) func(*Conn) {
	return func(c *Conn) {
		if v != nil {
			c.Validate = v
		}
	}
}

// Accept waits for and returns the next connection to the listener.
func (p *Listener) Accept() (net.Conn, error) {
	// Get the underlying connection
	conn, err := p.Listener.Accept()
	if err != nil {
		return nil, err
	}

	proxyHeaderPolicy := USE
	if p.Policy != nil {
		proxyHeaderPolicy, err = p.Policy(conn.RemoteAddr())
		if err != nil {
			// can't decide the policy, we can't accept the connection
			conn.Close()
			return nil, err
		}
	}

	newConn := NewConn(
		conn,
		WithPolicy(proxyHeaderPolicy),
		ValidateHeader(p.ValidateHeader),
	)
	return newConn, nil
}

// Close closes the underlying listener.
func (p *Listener) Close() error {
	return p.Listener.Close()
}

// Addr returns the underlying listener's network address.
func (p *Listener) Addr() net.Addr {
	return p.Listener.Addr()
}

// NewConn is used to wrap a net.Conn that may be speaking
// the proxy protocol into a proxyproto.Conn
func NewConn(conn net.Conn, opts ...func(*Conn)) *Conn {
	pConn := &Conn{
		bufReader: bufio.NewReader(conn),
		conn:      conn,
	}

	for _, opt := range opts {
		opt(pConn)
	}

	return pConn
}

// Read is check for the proxy protocol header when doing
// the initial scan. If there is an error parsing the header,
// it is returned and the socket is closed.
func (p *Conn) Read(b []byte) (int, error) {
	p.once.Do(func() {
		p.readErr = p.readHeader()
	})
	if p.readErr != nil {
		return 0, p.readErr
	}
	return p.bufReader.Read(b)
}

// Write wraps original conn.Write
func (p *Conn) Write(b []byte) (int, error) {
	return p.conn.Write(b)
}

// Close wraps original conn.Close
func (p *Conn) Close() error {
	return p.conn.Close()
}

// LocalAddr returns the address of the server if the proxy
// protocol is being used, otherwise just returns the address of
// the socket server. In case an error happens on reading the
// proxy header the original LocalAddr is returned, not the one
// from the proxy header even if the proxy header itself is
// syntactically correct.
func (p *Conn) LocalAddr() net.Addr {
	p.once.Do(func() { p.readErr = p.readHeader() })
	if p.header == nil || p.header.Command.IsLocal() || p.readErr != nil {
		return p.conn.LocalAddr()
	}

	return p.header.LocalAddr()
}

// RemoteAddr returns the address of the client if the proxy
// protocol is being used, otherwise just returns the address of
// the socket peer. In case an error happens on reading the
// proxy header the original RemoteAddr is returned, not the one
// from the proxy header even if the proxy header itself is
// syntactically correct.
func (p *Conn) RemoteAddr() net.Addr {
	p.once.Do(func() { p.readErr = p.readHeader() })
	if p.header == nil || p.header.Command.IsLocal() || p.readErr != nil {
		return p.conn.RemoteAddr()
	}

	return p.header.RemoteAddr()
}

// SetDeadline wraps original conn.SetDeadline
func (p *Conn) SetDeadline(t time.Time) error {
	return p.conn.SetDeadline(t)
}

// SetReadDeadline wraps original conn.SetReadDeadline
func (p *Conn) SetReadDeadline(t time.Time) error {
	return p.conn.SetReadDeadline(t)
}

// SetWriteDeadline wraps original conn.SetWriteDeadline
func (p *Conn) SetWriteDeadline(t time.Time) error {
	return p.conn.SetWriteDeadline(t)
}

func (p *Conn) readHeader() error {
	header, err := Read(p.bufReader)
	// For the purpose of this wrapper shamefully stolen from armon/go-proxyproto
	// let's act as if there was no error when PROXY protocol is not present.
	if err == ErrNoProxyProtocol {
		// but not if it is required that the connection has one
		if p.ProxyHeaderPolicy == REQUIRE {
			return err
		}

		return nil
	}

	// proxy protocol header was found
	if err == nil && header != nil {
		switch p.ProxyHeaderPolicy {
		case REJECT:
			// this connection is not allowed to send one
			return ErrSuperfluousProxyHeader
		case USE, REQUIRE:
			if p.Validate != nil {
				err = p.Validate(header)
				if err != nil {
					return err
				}
			}

			p.header = header
		}
	}

	return err
}
