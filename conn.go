package skytable_go

import (
	"bufio"
	"crypto/tls"
	"net"
	"time"

	"github.com/heliumbrain/skytable-go/marshal"
)

// Conn is a Client wrapping a single network connection which synchronously
// reads/writes data using the Skytable skyhash protocol.
//
// A Conn can be used directly as a Client - but for future versions a Pool will be implemented
type Conn interface {
	// Client method of a Conn is _not_ expected to be thread-safe with the
	// other methods of Conn, and merely calls the Action's Run method with
	// itself as the argument.
	Client

	// Encode and Decode may be called at the same time by two different
	// go-routines, but each should only be called once at a time (i.e. two
	// routines shouldn't call Encode at the same time, same with Decode).
	//
	// Encode and Decode should _not_ be called at the same time as Do.
	//
	// If either Encode or Decode encounter a net.Error the Conn will be
	// automatically closed.
	//
	// Encode is expected to encode an entire skyhash message, not a partial one.
	// In other words, when sending commands to skytable, Encode should only be
	// called once per command. Similarly, Decode is expected to decode an
	// entire skyhash response.
	Encode(marshal.Marshaler) error
	Decode(marshal.Unmarshaler) error

	// Returns the underlying network connection, as-is. Read, Write, and Close
	// should not be called on the returned Conn.
	NetConn() net.Conn
}

// ConnFunc is a function which returns an initialized, ready-to-be-used Conn.
// Functions like NewPool or NewCluster take in a ConnFunc in order to allow for
// things like calls to AUTH on each new connection, setting timeouts, custom
// Conn implementations, etc... See the package docs for more details.
type ConnFunc func(network, addr string) (Conn, error)

// DefaultConnFunc is a ConnFunc which will return a Conn for a skytable instance
// using sane defaults.
var DefaultConnFunc = func(network, addr string) (Conn, error) {
	return Dial(network, addr)
}

type connWrap struct {
	net.Conn
	brw *bufio.ReadWriter
}

// NewConn takes an existing net.Conn and wraps it to support the Conn interface
// of this package. The Read and Write methods on the original net.Conn should
// not be used after calling this method.
func NewConn(conn net.Conn) Conn {
	return &connWrap{
		Conn: conn,
		brw:  bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (cw *connWrap) Do(a Action) error {
	return a.Run(cw)
}

func (cw *connWrap) Encode(m marshal.Marshaler) error {
	if err := m.MarshalSkyhash(cw.brw); err != nil {
		return err
	}
	return cw.brw.Flush()
}

func (cw *connWrap) Decode(u marshal.Unmarshaler) error {
	return u.UnmarshalSkyhash(cw.brw.Reader)
}

func (cw *connWrap) NetConn() net.Conn {
	return cw.Conn
}

type dialOpts struct {
	connectTimeout, readTimeout, writeTimeout time.Duration
	authUser, authPass                        string
	selectDB                                  string
	useTLSConfig                              bool
	tlsConfig                                 *tls.Config
}

// DialOpt is an optional behavior which can be applied to the Dial function to
// effect its behavior, or the behavior of the Conn it creates.
type DialOpt func(*dialOpts)

// DialConnectTimeout determines the timeout value to pass into net.DialTimeout
// when creating the connection. If not set then net.Dial is called instead.
func DialConnectTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		do.connectTimeout = d
	}
}

// DialReadTimeout determines the deadline to set when reading from a dialed
// connection. If not set then SetReadDeadline is never called.
func DialReadTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		do.readTimeout = d
	}
}

// DialWriteTimeout determines the deadline to set when writing to a dialed
// connection. If not set then SetWriteDeadline is never called.
func DialWriteTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		do.writeTimeout = d
	}
}

// DialTimeout is the equivalent to using DialConnectTimeout, DialReadTimeout,
// and DialWriteTimeout all with the same value.
func DialTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		DialConnectTimeout(d)(do)
		DialReadTimeout(d)(do)
		DialWriteTimeout(d)(do)
	}
}

type timeoutConn struct {
	net.Conn
	readTimeout, writeTimeout time.Duration
}

func (tc *timeoutConn) Read(b []byte) (int, error) {
	if tc.readTimeout > 0 {
		err := tc.Conn.SetReadDeadline(time.Now().Add(tc.readTimeout))
		if err != nil {
			return 0, err
		}
	}
	return tc.Conn.Read(b)
}

func (tc *timeoutConn) Write(b []byte) (int, error) {
	if tc.writeTimeout > 0 {
		err := tc.Conn.SetWriteDeadline(time.Now().Add(tc.writeTimeout))
		if err != nil {
			return 0, err
		}
	}
	return tc.Conn.Write(b)
}

var defaultDialOpts = []DialOpt{
	DialTimeout(10 * time.Second),
}

// Dial is a ConnFunc which creates a Conn using net.Dial and NewConn. It takes
// in a number of options which can overwrite its default behavior as well.
//
// In place of a host:port address, Dial also accepts a URI, as per:
// 	https://www.iana.org/assignments/uri-schemes/prov/skytable
// If the URI has an AUTH password or db specified Dial will attempt to perform
// the AUTH and/or SELECT as well.
//
// If either DialAuthPass or DialSelectDB is used it overwrites the associated
// value passed in by the URI.
//
// The default options Dial uses are:
//
//	DialTimeout(10 * time.Second)
//
func Dial(network, addr string, opts ...DialOpt) (Conn, error) {
	var do dialOpts
	for _, opt := range defaultDialOpts {
		opt(&do)
	}
	for _, opt := range opts {
		opt(&do)
	}

	var netConn net.Conn
	var err error
	dialer := net.Dialer{}
	if do.connectTimeout > 0 {
		dialer.Timeout = do.connectTimeout
	}
	if do.useTLSConfig {
		netConn, err = tls.DialWithDialer(&dialer, network, addr, do.tlsConfig)
	} else {
		netConn, err = dialer.Dial(network, addr)
	}

	if err != nil {
		return nil, err
	}

	// If the netConn is a net.TCPConn (or some wrapper for it) and so can have
	// keepalive enabled, do so with a sane (though slightly aggressive)
	// default.
	{
		type keepaliveConn interface {
			SetKeepAlive(bool) error
			SetKeepAlivePeriod(time.Duration) error
		}

		if kaConn, ok := netConn.(keepaliveConn); ok {
			if err = kaConn.SetKeepAlive(true); err != nil {
				netConn.Close()
				return nil, err
			} else if err = kaConn.SetKeepAlivePeriod(10 * time.Second); err != nil {
				netConn.Close()
				return nil, err
			}
		}
	}

	conn := NewConn(&timeoutConn{
		readTimeout:  do.readTimeout,
		writeTimeout: do.writeTimeout,
		Conn:         netConn,
	})

	return conn, nil
}
