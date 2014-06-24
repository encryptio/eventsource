package eventsource

import (
	"fmt"
	"io"
	"strings"
)

type Event struct {
	Data  string
	ID    string
	Event string
}

func (e Event) writeTo(w io.Writer) error {
	if e.Event != "" {
		_, err := fmt.Fprintf(w, "event:%s\n", e.Event)
		if err != nil {
			return err
		}
	}

	for _, str := range strings.Split(e.Data, "\n") {
		_, err := fmt.Fprintf(w, "data:%s\n", str)
		if err != nil {
			return err
		}
	}

	if e.ID != "" {
		_, err := fmt.Fprintf(w, "id:%s\n", e.ID)
		if err != nil {
			return err
		}
	}

	_, err := w.Write([]byte{'\n'})
	return err
}
