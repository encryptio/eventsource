package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/encryptio/eventsource"
)

func stream(r *http.Request, ch chan<- eventsource.Event, done <-chan struct{}) {
	n := 0
	for {
		data := fmt.Sprintf("%v: %s", n, time.Now().String())

		select {
		case ch <- eventsource.Event{data, "", ""}:
		case <-done:
			return
		}

		time.Sleep(100 * time.Millisecond)
		n++
	}
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	w.Write([]byte(`
<!DOCTYPE HTML>
<html>
<head>
</head>
<body>
<div id="status">Connecting</div>
<div id="data"></div>
<script>
var source = new EventSource("/stream");
source.addEventListener('message', function (evt) {
    var el = document.getElementById("data");
    el.innerHTML = evt.data;
    console.log("message");
});
source.addEventListener('open', function (evt) {
    var el = document.getElementById("status");
    el.innerHTML = "Open";
    console.log("open");
});
source.addEventListener('error', function (evt) {
    var el = document.getElementById("status");
    el.innerHTML = "Disconnected, state is "+source.readyState;
});
console.log("going");
</script>
</body>
</html>
`))
}

func main() {
	http.Handle("/stream", &eventsource.Handler{Stream: stream})
	http.HandleFunc("/page", pageHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
