package memnet

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type addr struct {
	address string
}

func (a addr) Network() string { return a.address }

func (a addr) String() string { return a.address }

type netErrTimeout struct {
	error
}

func (net netErrTimeout) Timeout() bool {
	return true
}

func (net netErrTimeout) Temporary() bool {
	return false
}

var (
	errClosed            = fmt.Errorf("closed")
	errTimeout net.Error = netErrTimeout{error: fmt.Errorf("i/o timeout")}
)

type ringBuff struct {
	buff   []byte
	r, w   int
	mu     sync.Mutex
	rdwait sync.Cond
	wrwait sync.Cond

	rdtimer *time.Timer
	wrtimer *time.Timer

	rdtimeout bool
	wrtimeout bool

	closed      bool
	writeClosed bool
}

func (rb *ringBuff) empty() bool {
	return rb.r == len(rb.buff)
}

func (rb *ringBuff) full() bool {
	return rb.r == rb.w && rb.r < len(rb.buff)
}

func (rb *ringBuff) Close() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		return io.ErrClosedPipe
	}

	rb.closed = true

	// Signal all blocked readers and writers
	rb.rdwait.Broadcast()
	rb.wrwait.Broadcast()
	return nil
}

func (rb *ringBuff) closeWrite() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		return io.ErrClosedPipe
	}

	rb.writeClosed = true

	// Signal all blocked readers and writers
	rb.rdwait.Broadcast()
	rb.wrwait.Broadcast()
	return nil
}

func (rb *ringBuff) Write(data []byte) (int, error) {
	rb.wrwait.L.Lock()
	defer rb.wrwait.L.Unlock()

	if rb.closed {
		return 0, io.ErrClosedPipe
	}

	var n int

	for len(data) > 0 {
		// Wait until ringBuff drains
		for {

			if rb.closed || rb.writeClosed {
				return 0, io.ErrClosedPipe
			}

			if !rb.full() {
				break
			}

			if rb.wrtimeout {
				return 0, errTimeout
			}

			rb.wrwait.Wait()
		}

		endPos := cap(rb.buff)
		if rb.w < rb.r {
			endPos = rb.r
		}

		cn := copy(rb.buff[rb.w:endPos], data)
		n += cn
		rb.w += cn

		// Reslice the input buffer
		data = data[cn:]

		// It is possible that current len(rb.buff) is less than
		// new rb.w we have to reslice the buff to make new writes
		// available fro read
		if rb.w > len(rb.buff) {
			rb.buff = rb.buff[:rb.w]
		}

		if rb.w == cap(rb.buff) {
			rb.w = 0
		}

		if !rb.empty() {
			// Ring buffer is not empty, signal readers
			rb.rdwait.Signal()
		}
	}

	return n, nil
}

func (rb *ringBuff) Read(data []byte) (int, error) {
	rb.rdwait.L.Lock()
	defer rb.rdwait.L.Unlock()

	for {

		if rb.closed {
			return 0, io.ErrClosedPipe
		}

		// Wait till ring buffer gets filled up
		if !rb.empty() {
			break
		}

		if rb.rdtimeout {
			return 0, errTimeout
		}

		if rb.writeClosed {
			return 0, io.EOF
		}

		rb.rdwait.Wait()
	}

	//reads are possible in window of [rb.r, len(rb.buff))

	n := copy(data, rb.buff[rb.r:len(rb.buff)])
	rb.r += n

	if rb.r == cap(rb.buff) {
		rb.r = 0

		// Adjust the reading window ring buffer
		// new window should be [0, rb.w)
		rb.buff = rb.buff[:rb.w]
	}

	if !rb.full() {
		// Ring buffer is not full, signal writers
		rb.wrwait.Signal()
	}

	return n, nil
}

func newRingBuff(size int) *ringBuff {
	b := make([]byte, 0, size)

	rb := &ringBuff{}
	rb.buff = b
	rb.rdwait.L = &rb.mu
	rb.wrwait.L = &rb.mu
	rb.rdtimer = time.AfterFunc(0, func() {})
	rb.wrtimer = time.AfterFunc(0, func() {})
	return rb
}

type conn struct {
	r io.Reader
	w io.Writer
}

func (c *conn) LocalAddr() net.Addr {
	return addr{}
}

func (c *conn) RemoteAddr() net.Addr {
	return addr{}
}

func (c *conn) SetReadDeadline(t time.Time) error {
	rb := c.r.(*ringBuff)
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.rdtimer.Stop()

	// If t is not initiliazed
	if t.IsZero() {
		rb.rdtimeout = false
		return nil
	}

	rb.rdtimer = time.AfterFunc(time.Until(t), func() {
		rb.mu.Lock()
		defer rb.mu.Unlock()
		rb.rdtimeout = true
		rb.rdwait.Broadcast()
	})

	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	rb := c.r.(*ringBuff)
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.wrtimer.Stop()

	// If t is not initialized
	if t.IsZero() {
		rb.wrtimeout = false
		return nil
	}

	rb.wrtimer = time.AfterFunc(time.Until(t), func() {
		rb.mu.Lock()
		defer rb.mu.Unlock()

		rb.wrtimeout = true
		rb.wrwait.Broadcast()
	})

	return nil
}

func (c *conn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (c *conn) Write(b []byte) (int, error) {
	return c.w.Write(b)
}

func (c *conn) Close() error {
	err := c.r.(*ringBuff).Close()
	if err != nil {
		return fmt.Errorf("closing a closed connection")
	}
	err = c.w.(*ringBuff).closeWrite()
	if err != nil {
		return fmt.Errorf("closing a closed connection")
	}
	return nil
}

// Listener satisfies net.Listener
type Listener struct {
	mu     sync.Mutex
	bsz    int
	connCh chan net.Conn
	done   chan struct{}
	addr   net.Addr
}

func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case <-l.done:
		return io.ErrClosedPipe
	default:
		close(l.done)
	}
	return nil
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, io.ErrClosedPipe
	case c := <-l.connCh:
		return c, nil
	}
}

func (l *Listener) Addr() net.Addr { return l.addr }

// Dial returns a client side connection to the attached to thre reciever.
func (l *Listener) Dial() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, io.ErrClosedPipe
	default:
		p1 := newRingBuff(l.bsz)
		p2 := newRingBuff(l.bsz)

		l.connCh <- &conn{p1, p2}
		return &conn{p2, p1}, nil
	}
}

// Listen returns a *Listener which can queue connQSize number of
// new connections till it blocks the call to Accept() and have
//transport buffer size of transBuffSize
func Listen(connQSize, transBuffSize int, _addr string) (*Listener, error) {
	l := &Listener{
		sync.Mutex{},
		transBuffSize,
		make(chan net.Conn, connQSize),
		make(chan struct{}),
		addr{_addr},
	}

	return l, nil
}
