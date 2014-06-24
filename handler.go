package eventsource

import (
	"fmt"
	"net/http"
)

const DefaultRetryInterval = 5000

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
	// between browser retries on a failed connection. It defaults to DefaultRetryInterval
	// if negative or not set.
	RetryInterval int
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

	_, err = buf.Write([]byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: identity\r\nContent-Type: text/event-stream\r\nConnection: close\r\n\r\n"))
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

	for evt := range ch {
		err = evt.writeTo(buf)
		if err != nil {
			return
		}

		err = buf.Flush()
		if err != nil {
			return
		}
	}
}
