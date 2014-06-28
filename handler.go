package eventsource

import (
	"fmt"
	"net/http"
	"time"
)

const DefaultRetryInterval = 5000
const DefaultKeepAliveInterval = 50000 * time.Millisecond

var keepAliveMessage = []byte(":\n")
var header = []byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: identity\r\nContent-Type: text/event-stream\r\nConnection: close\r\nCache-control: no-cache\r\n\r\n")

// Handler implements the event stream interface with a user-given stream function.
// When a request is recieved by ServeHTTP, it writes the proper header, sends the
// RetryInterval, and calls Stream in another goroutine.
//
// Example usage: http.Handle("/stream", &eventsource.Handler{streamFn, 10000})
type Handler struct {
	// Stream is called when a request comes in. It is expected to send a series
	// of events on ch, and to exit when the done channel is closed.
	Stream func(r *http.Request, ch chan<- Event, done <-chan struct{})

	// RetryInterval corresponds to the "retry:" field and is a number of milliseconds
	// between browser retries on a failed connection. Defaults to DefaultRetryInterval
	// if <= 0.
	RetryInterval int

	// KeepAliveInterval defines how fast to send empty ":\n" comment lines when no events
	// are being sent. This can detect dead connections and keep live ones live. Defaults
	// to DefaultKeepAliveInterval if <= 0.
	KeepAliveInterval time.Duration
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	done := make(chan struct{})
	defer close(done)

	// panic if we can't hijack
	hj := w.(http.Hijacker)

	conn, buf, err := hj.Hijack()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	defer buf.Flush()

	// work around https://code.google.com/p/go/issues/detail?id=8296
	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Time{})

	_, err = buf.Write(header)
	if err != nil {
		return
	}

	retry := DefaultRetryInterval
	if h.RetryInterval > 0 {
		retry = h.RetryInterval
	}
	_, err = fmt.Fprintf(buf, "retry:%d\n", retry)
	if err != nil {
		return
	}

	err = buf.Flush()
	if err != nil {
		return
	}

	ch := make(chan Event)
	go h.Stream(r, ch, done)

	keepAlive := DefaultKeepAliveInterval
	if h.KeepAliveInterval > 0 {
		keepAlive = h.KeepAliveInterval
	}

	timer := time.NewTimer(keepAlive)
	defer timer.Stop()

	for {
		timer.Reset(keepAlive)

		select {
		case <-timer.C:
			_, err := buf.Write(keepAliveMessage)
			if err != nil {
				return
			}

		case evt, ok := <-ch:
			if !ok {
				return
			}

			err = evt.writeTo(buf)
			if err != nil {
				return
			}
		}

		err = buf.Flush()
		if err != nil {
			return
		}
	}
}
