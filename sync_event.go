package gonedrive

import "fmt"

type SyncEventFn func(event SyncEvent)

type SyncEvent interface {
	fmt.Stringer
}

// Begin upload/download.
// This event is sent whenever a download or upload is started.
// A corrosponding SyncEventEnd event will also be sent eventually.
// While the file is uploading/downloading, any number of
// SyncEventProgress events.
type SyncEventBegin struct {
	LocalPath  string
	RemotePath string
	IsUpload   bool
}

func (event SyncEventBegin) String() string {
	action := "downloading"
	fname := event.RemotePath
	if event.IsUpload {
		action = "uploading"
		fname = event.LocalPath
	}

	return fmt.Sprintf(
		"%s \"%s\"",
		action,
		fname,
	)
}

// End upload/download.
// Once this event has been sent, no more events will be sent for this file.
type SyncEventEnd struct {
	LocalPath  string
	RemotePath string
	IsUpload   bool
	Success    bool
}

func (event SyncEventEnd) String() string {
	action := "download"
	fname := event.RemotePath
	if event.IsUpload {
		action = "upload"
		fname = event.LocalPath
	}

	status := "finished"
	if !event.Success {
		status = "failed"
	}

	return fmt.Sprintf(
		"%s of \"%s\" %s",
		action,
		fname,
		status,
	)
}

// Error
type SyncEventError struct {
	LocalPath  string
	RemotePath string
	Err        error
}

func (event SyncEventError) String() string {
	return fmt.Sprintf(
		"error syncing \"%s\": %s",
		event.RemotePath,
		event.Err,
	)
}

// Progress event.
type SyncEventProgress struct {
	LocalPath  string
	RemotePath string
	IsUpload   bool
	Size       int64
	Progress   int64
}

func (event SyncEventProgress) String() string {
	action := "downloaded"
	fname := event.RemotePath
	if event.IsUpload {
		action = "uploaded"
		fname = event.LocalPath
	}

	return fmt.Sprintf(
		"%s %d/%d bytes of \"%s\"",
		action,
		event.Progress,
		event.Size,
		fname,
	)
}

// Skipped downloading file.
type SyncEventSkip struct {
	LocalPath  string
	RemotePath string
}

func (event SyncEventSkip) String() string {
	return fmt.Sprintf(
		"skipping \"%s\", local file up to date",
		event.RemotePath,
	)
}

// Deleted local file.
type SyncEventDelete struct {
	LocalPath string
}

func (event SyncEventDelete) String() string {
	return fmt.Sprintf(
		"deleted \"%s\"",
		event.LocalPath,
	)
}
