package raw

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/12end/request/raw/client"
)

// Dialer can dial a remote HTTP server.
type Dialer interface {
	// Dial dials a remote http server returning a Conn.
	Dial(protocol, addr string, options *Options) (Conn, error)
	// Dial dials a remote http server with timeout returning a Conn.
	DialTimeout(protocol, addr string, timeout time.Duration, options *Options) (Conn, error)
}

type dialer struct {
	sync.Mutex                   // protects following fields
	conns      map[string][]Conn // maps addr to a, possibly empty, slice of existing Conns
}

func (d *dialer) Dial(protocol, addr string, options *Options) (Conn, error) {
	return d.dialTimeout(protocol, addr, 0, options)
}

func (d *dialer) DialTimeout(protocol, addr string, timeout time.Duration, options *Options) (Conn, error) {
	return d.dialTimeout(protocol, addr, timeout, options)
}

func (d *dialer) dialTimeout(protocol, addr string, timeout time.Duration, options *Options) (Conn, error) {
	d.Lock()
	if d.conns == nil {
		d.conns = make(map[string][]Conn)
	}
	if c, ok := d.conns[addr]; ok {
		if len(c) > 0 {
			conn := c[0]
			c[0] = c[len(c)-1]
			d.Unlock()
			return conn, nil
		}
	}
	d.Unlock()
	c, err := clientDial(protocol, addr, timeout, options)
	return &conn{
		Client: client.NewClient(c),
		Conn:   c,
		dialer: d,
	}, err
}

func clientDial(protocol, addr string, timeout time.Duration, options *Options) (net.Conn, error) {

	// http
	if protocol == "http" {
		if timeout > 0 {
			return net.DialTimeout("tcp", addr, timeout)
		}
		return net.Dial("tcp", addr)
	}

	// https
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if options.SNI != "" {
		tlsConfig.ServerName = options.SNI
	}
	return tls.Dial("tcp", addr, tlsConfig)
}

// TlsHandshake tls handshake on a plain connection
func TlsHandshake(conn net.Conn, addr string, timeout time.Duration) (net.Conn, error) {
	colonPos := strings.LastIndex(addr, ":")
	if colonPos == -1 {
		colonPos = len(addr)
	}
	hostname := addr[:colonPos]

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         hostname,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}
	return tlsConn, nil
}

// Conn is an interface implemented by a connection
type Conn interface {
	client.Client
	io.Closer

	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	Release()
}

type conn struct {
	client.Client
	net.Conn
	*dialer
}

func (c *conn) Release() {
	c.dialer.Lock()
	defer c.dialer.Unlock()
	addr := c.Conn.RemoteAddr().String()
	c.dialer.conns[addr] = append(c.dialer.conns[addr], c)
}
