package ui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/restic/restic/internal/archiver"
)

type HttpRestore struct {
	enabled   bool
	State     int
	r         *Restore
	ScanStats archiver.ScanStats
	Snapshot  string
	startTime time.Time

	url      string
	interval int
	token    string
}

func (h HttpRestore) send(message httpMessage) {
	if message.Status == "none" {
		return
	}
	j, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	_, err = http.Post(h.url, "text/json", bytes.NewReader(j))
	if err != nil {
		panic(err)
	}
}

func (h HttpRestore) newMessage() httpMessage {
	msg := httpMessage{
		Token:     h.token,
		Action:    "restore",
		StartTime: h.startTime,
		Snapshot:  h.Snapshot,
		PID:       os.Getpid(),
	}
	switch h.State {
	case HTTP_NONE:
		msg.Status = "none" // Should never occur
	case HTTP_READING_INDEX:
		msg.Status = "indexing"
	case HTTP_SCANNING_DATA:
		msg.Status = "scanning"
	case HTTP_DOING_RESTORE:
		msg.Status = "doing_restore"
	case HTTP_DONE:
		msg.Status = "done"
	default:
		panic("Unexpected state " + strconv.Itoa(h.State))
	}
	msg.Successful = true
	msg.ErrorMsg = ""
	return msg
}

func (h HttpRestore) Error(err error) {
	if !h.enabled { return }
	msg := h.newMessage()
	msg.Successful = false
	msg.ErrorMsg = err.Error()
	h.send(msg)
}

func (h HttpRestore) SendUpdate() {
	if !h.enabled { return }
	msg := h.newMessage()
	msg.SecsElapsed = int64(time.Since(h.r.start).Seconds())
	if h.State == HTTP_SCANNING_DATA {
		msg.FilesProcessed = h.ScanStats.Files
		msg.BytesProcessed = h.ScanStats.Bytes
	} else {
		msg.FilesProcessed = h.r.processed.Files
		msg.BytesProcessed = h.r.processed.Bytes
	}
	msg.DirsNew = h.r.summary.Dirs.New
	msg.NumErrors = h.r.errors
	if (h.r.total.Files != 0 || h.r.total.Dirs != 0) && h.r.eta > 0 && h.r.processed.Bytes < h.r.total.Bytes {
		msg.HasETA = true
		msg.ETA = h.r.eta
	}
	h.send(msg)
}

func (h *HttpRestore) SendDone() {
	if !h.enabled { return }
	h.State = HTTP_DONE
	msg := h.newMessage()
	h.State = HTTP_NONE
	msg.FilesNew = h.r.summary.Files.New
	msg.DirsNew = h.r.summary.Dirs.New
	msg.SecsElapsed = int64(time.Since(h.r.start).Seconds())
	msg.FilesProcessed = h.r.processed.Files
	msg.BytesProcessed = h.r.processed.Bytes
	msg.NumErrors = h.r.errors
	h.send(msg)
}

func NewHttpRestore(r *Restore, url string, interval int, token string) *HttpRestore {
	if url == "" {
		return &HttpRestore{}
	}
	ticker := time.NewTicker(time.Duration(interval * int(time.Second)))
	instance := HttpRestore{
		enabled:   true,
		url:       url,
		interval:  interval,
		token:     token,
		r:         r,
		startTime: time.Now(),
	}
	go func() {
		for range ticker.C {
			instance.SendUpdate()
		}
	}()
	return &instance
}
