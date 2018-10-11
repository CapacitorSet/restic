package ui

import "time"

const (
	HTTP_NONE = iota
	HTTP_READING_INDEX
	HTTP_SCANNING_DATA
	HTTP_DOING_BACKUP
	HTTP_DOING_RESTORE
	HTTP_DONE
)

type httpMessage struct {
	Token      string    `json:"token"`
	Action     string    `json:"action"` // backup o restore
	PID        int       `json:"pid"`
	Status     string    `json:"status"`
	Snapshot   string    `json:"snapshot"`
	StartTime  time.Time `json:"start-time"`
	Successful bool      `json:"successful"`
	ErrorMsg   string    `json:"error_msg"`

	SecsElapsed    int64  `json:"elapsed"`
	FilesProcessed uint   `json:"files_processed"`
	BytesProcessed uint64 `json:"bytes_processed"`
	NumErrors      uint   `json:"errors"`

	// The fields below are only valid during a backup.
	ETA    uint64 `json:"eta"`
	HasETA bool   `json:"has_eta"` // True if no ETA is available

	// The fields below are only valid for status=done.
	FilesNew        uint `json:"files_new"`
	FilesChanged    uint `json:"files_changed"`
	FilesUnmodified uint `json:"files_unmodified"`
	DirsNew         uint `json:"dirs_new"`
	DirsChanged     uint `json:"dirs_changed"`
	DirsUnmodified  uint `json:"dirs_unmodified"`
}
