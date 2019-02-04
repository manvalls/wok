package wok

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/manvalls/wit"
)

func (h Handler) handleWS(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()

	rootCtx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	mutex := sync.Mutex{}
	mapsLock := sync.Mutex{}
	cancels := make(map[string]context.CancelFunc)
	inChannels := make(map[string]chan url.Values)

	cleanup := func(id string) {
		mapsLock.Lock()
		defer mapsLock.Unlock()

		cancel, ok := cancels[id]
		if ok {
			cancel()
			delete(cancels, id)

			mutex.Lock()
			defer mutex.Unlock()

			conn.WriteMessage(websocket.TextMessage, []byte("DONE "+id+"\r\n"))
		}

		chIn, ok := inChannels[id]
		if ok {
			close(chIn)
			delete(inChannels, id)
		}
	}

	for {
		_, r, err := conn.NextReader()
		if err != nil {
			return
		}

		reader := bufio.NewReader(r)
		command, err := reader.ReadSlice(' ')
		if err != nil {
			return
		}

		command = command[:len(command)-1]

		idBytes, err := reader.ReadSlice('\n')
		if err != nil {
			return
		}

		if len(idBytes) > 1 && idBytes[len(idBytes)-2] == '\r' {
			idBytes = idBytes[:len(idBytes)-2]
		} else {
			idBytes = idBytes[:len(idBytes)-1]
		}

		id := string(idBytes)

		switch string(command) {
		case "REQUEST":
			cleanup(id)
			req, err := http.ReadRequest(reader)
			if err != nil {
				return
			}

			ctx, cancel := context.WithCancel(rootCtx)
			chOut := make(chan wit.Action)
			chIn := make(chan url.Values, h.InputBuffer)

			req = req.WithContext(ctx)

			mapsLock.Lock()
			cancels[id] = cancel
			inChannels[id] = chIn
			mapsLock.Unlock()

			go func() {
				defer cleanup(id)

				for {

					select {
					case action, ok := <-chOut:
						if !ok {
							return
						}

						mutex.Lock()

						nw, werr := conn.NextWriter(websocket.TextMessage)
						if werr != nil {
							mutex.Unlock()
							return
						}

						nw.Write([]byte("APPLY " + id + "\r\n"))
						rerr := wit.NewJSONRenderer(action).Render(nw)
						nw.Close()

						if rerr != nil {
							mutex.Unlock()
							return
						}

						mutex.Unlock()

					case <-ctx.Done():
						return
					}
				}
			}()

			go func() {
				w := &wsResponseWriter{
					Conn:   conn,
					wsMux:  &mutex,
					header: http.Header{},
					id:     id,
				}

				h.serve(w, req, chIn, chOut, func() {
					w.WriteHeader(http.StatusOK)
					if w.werr != nil {
						w.wc.Close()
					}
					w.wsMux.Unlock()
				})

				cleanup(id)
			}()
		case "CLOSE":
			cleanup(id)
		case "EVENT":
			data, _ := ioutil.ReadAll(reader)

			go func() {
				mapsLock.Lock()
				defer mapsLock.Unlock()

				chIn, ok := inChannels[id]
				if ok {
					params, _ := url.ParseQuery(string(data))
					select {
					case chIn <- params:
					default:
					}
				}
			}()
		}
	}

}

type wsResponseWriter struct {
	sync.Mutex
	*websocket.Conn
	wsMux  *sync.Mutex
	wc     io.WriteCloser
	werr   error
	header http.Header
	id     string
}

func (w *wsResponseWriter) Header() http.Header {
	return w.header
}

func (w *wsResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	if w.werr != nil {
		return 0, w.werr
	}

	return w.wc.Write(p)
}

func (w *wsResponseWriter) WriteHeader(statusCode int) {
	w.Lock()
	defer w.Unlock()

	if w.wc == nil && w.werr == nil {
		w.wsMux.Lock()
		w.wc, w.werr = w.NextWriter(websocket.TextMessage)
		if w.werr == nil {
			w.wc.Write([]byte("RESPONSE " + w.id + "\r\n"))
			w.wc.Write([]byte("HTTP/1.0 " + strconv.FormatInt(int64(statusCode), 10)))

			statusText := http.StatusText(statusCode)
			if statusText != "" {
				w.wc.Write([]byte(" " + statusText))
			}

			w.wc.Write([]byte("\r\n"))

			for key, values := range w.header {
				for _, value := range values {
					w.wc.Write([]byte(key + ": " + value + "\r\n"))
				}
			}

			w.wc.Write([]byte("\r\n"))
		}
	}
}
