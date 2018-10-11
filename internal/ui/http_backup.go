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

type HttpBackup struct {
	enabled   bool
	State     int
	b         *Backup
	ScanStats archiver.ScanStats
	startTime time.Time

	url      string
	interval int
	token    string
}

func (h HttpBackup) send(message httpMessage) {
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

func (h HttpBackup) newMessage() httpMessage {
	msg := httpMessage{
		Token:     h.token,
		Action:    "backup",
		StartTime: h.startTime,
		PID:       os.Getpid(),
	}
	switch h.State {
	case HTTP_NONE:
		msg.Status = "none" // Should never occur
	case HTTP_READING_INDEX:
		msg.Status = "indexing"
	case HTTP_SCANNING_DATA:
		msg.Status = "scanning"
	case HTTP_DOING_BACKUP:
		msg.Status = "doing_backup"
	case HTTP_DONE:
		msg.Status = "done"
	default:
		panic("Unexpected state " + strconv.Itoa(h.State))
	}
	msg.Successful = true
	msg.ErrorMsg = ""
	return msg
}

func (h HttpBackup) Error(err error) {
	if !h.enabled { return }
	msg := h.newMessage()
	msg.Successful = false
	msg.ErrorMsg = err.Error()
	h.send(msg)
}

func (h HttpBackup) SendUpdate() {
	if !h.enabled { return }
	msg := h.newMessage()
	msg.SecsElapsed = int64(time.Since(h.b.start).Seconds())
	if h.State == HTTP_SCANNING_DATA {
		msg.FilesProcessed = h.ScanStats.Files
		msg.BytesProcessed = h.ScanStats.Bytes
	} else {
		msg.FilesProcessed = h.b.processed.Files
		msg.BytesProcessed = h.b.processed.Bytes
	}
	msg.NumErrors = h.b.errors
	if (h.b.total.Files != 0 || h.b.total.Dirs != 0) && h.b.eta > 0 && h.b.processed.Bytes < h.b.total.Bytes {
		msg.HasETA = true
		msg.ETA = h.b.eta
	}
	h.send(msg)
}

func (h *HttpBackup) SendDone(snapshot string) {
	if !h.enabled { return }
	h.State = HTTP_DONE
	msg := h.newMessage()
	h.State = HTTP_NONE
	msg.Snapshot = snapshot
	msg.FilesNew = h.b.summary.Files.New
	msg.FilesChanged = h.b.summary.Files.Changed
	msg.FilesUnmodified = h.b.summary.Files.Unchanged
	msg.DirsNew = h.b.summary.Dirs.New
	msg.DirsChanged = h.b.summary.Dirs.Changed
	msg.DirsUnmodified = h.b.summary.Dirs.Unchanged
	msg.SecsElapsed = int64(time.Since(h.b.start).Seconds())
	msg.FilesProcessed = h.b.processed.Files
	msg.BytesProcessed = h.b.processed.Bytes
	msg.NumErrors = h.b.errors
	h.send(msg)
}

func NewHttpBackup(b *Backup, url string, interval int, token string) *HttpBackup {
	if url == "" {
		return &HttpBackup{}
	}
	ticker := time.NewTicker(time.Duration(interval * int(time.Second)))
	instance := HttpBackup{
		enabled:   true,
		url:       url,
		interval:  interval,
		token:     token,
		b:         b,
		startTime: time.Now(),
	}
	go func() {
		for range ticker.C {
			instance.SendUpdate()
		}
	}()
	return &instance
}
