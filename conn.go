package icapclient

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ICAPConn is the one responsible for driving the transport layer operations. We have to explicitly deal with the connection because the ICAP protocol is aware of keep alive and reconnects.
type ICAPConn struct {
	tcp net.Conn
	mu  sync.Mutex
}

// NewICAPConn creates a new connection to the icap server
func NewICAPConn() (*ICAPConn, error) {
	return &ICAPConn{}, nil
}

// Connect connects to the icap server
func (c *ICAPConn) Connect(ctx context.Context, address string, timeout time.Duration) error {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}

	c.tcp = conn

	if dialer.Timeout == 0 {
		return nil
	}

	deadline := time.Now().UTC().Add(dialer.Timeout)

	if err := c.tcp.SetReadDeadline(deadline); err != nil {
		return err
	}

	if err := c.tcp.SetWriteDeadline(deadline); err != nil {
		return err
	}

	return nil
}

// Send sends a request to the icap server
func (c *ICAPConn) Send(in []byte) (*Response, error) {
	if !c.ok() {
		return nil, syscall.EINVAL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	errChan := make(chan error)
	resChan := make(chan *Response)

	go func() {
		// send the message to the server
		_, err := c.tcp.Write(in)
		if err != nil {
			errChan <- err
		}
	}()

	go func() {
		data := make([]byte, 0)

		for {
			tmp := make([]byte, 1096)

			// read the response from the server
			n, err := c.tcp.Read(tmp)

			// something went wrong, exit the loop and send the error
			if err != nil && err != io.EOF {
				errChan <- err
			}

			// EOF detected, an entire message is received
			if err == io.EOF || n == 0 {
				break
			}

			data = append(data, tmp[:n]...)

			// explicitly breaking because the Read blocks for 100 continue message
			// fixMe: still unclear why this is happening, find out and fix it
			if string(data) == icap100ContinueMsg {
				break
			}

			// EOF detected, 0 Double crlf indicates the end of the message
			if strings.HasSuffix(string(data), "0\r\n\r\n") {
				break
			}

			// EOF detected, 204 no modifications and Double crlf indicate the end of the message
			if strings.Contains(string(data), icap204NoModsMsg) {
				break
			}
		}

		resp, err := readResponse(bufio.NewReader(strings.NewReader(string(data))))
		if err != nil {
			errChan <- err
		}

		resChan <- resp
	}()

	select {
	case err := <-errChan:
		return nil, err
	case res := <-resChan:
		return res, nil
	}
}

// Close closes the tcp connection
func (c *ICAPConn) Close() error {
	if !c.ok() {
		return syscall.EINVAL
	}

	return c.tcp.Close()
}

func (c *ICAPConn) ok() bool { return c != nil && c.tcp != nil }
