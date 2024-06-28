package gonedrive

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/sukus21/gonedrive/quickxor"
)

var ErrSyncLocalDirectory = errors.New("local file is directory")
var ErrSyncRemoteDirectory = errors.New("remote file is directory")

type SyncFile struct {
	// Full file path
	FileName string

	// File size, not relevant for directories
	Size int64

	// Is this a directory?
	IsDir bool
}

type SyncFilterFn func(file SyncFile) bool

type syncContext struct {
	t          *GraphToken
	mux        sync.Mutex
	wg         sync.WaitGroup
	c          chan *DriveItem
	localFiles map[string]SyncFile
	filterFn   SyncFilterFn
	eventFn    SyncEventFn
	remotePath string
	localPath  string
}

func (ctx *syncContext) addItem(item *DriveItem) {
	ctx.wg.Add(1)
	ctx.c <- item
}

func (ctx *syncContext) syncQueue() {
	for item := range ctx.c {
		ctx.handleItem(item)
	}
}

func (ctx *syncContext) sendEvent(event SyncEvent) {
	if ctx.eventFn != nil {
		ctx.eventFn(event)
	}
}

func (ctx *syncContext) handleItem(item *DriveItem) {
	defer ctx.wg.Done()

	// Find local file in map
	ctx.mux.Lock()
	localFile, exists := ctx.localFiles[item.Name]
	delete(ctx.localFiles, item.Name)
	ctx.mux.Unlock()

	// I'll be using these
	remotePath := path.Join(ctx.remotePath, item.Name)
	localPath := filepath.Join(ctx.localPath, item.Name)

	// Cannot sync directories at the moment
	if item.IsDir() {
		ctx.sendEvent(&SyncEventError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Err:        ErrSyncRemoteDirectory,
		})
		return
	}

	// Handle existing local file
	if exists {
		// Cannot sync directories
		if localFile.IsDir {
			ctx.sendEvent(&SyncEventError{
				LocalPath:  localPath,
				RemotePath: remotePath,
				Err:        ErrSyncLocalDirectory,
			})
			return
		}

		// Is local file identical?
		if ctx.syncFilesIdentical(localFile, item) {
			ctx.sendEvent(&SyncEventSkip{
				LocalPath:  localPath,
				RemotePath: remotePath,
			})
			return
		}

		// No they are not, remove local file
		if err := os.Remove(localPath); err != nil {
			ctx.sendEvent(&SyncEventError{
				LocalPath:  localPath,
				RemotePath: remotePath,
				Err:        err,
			})
			return
		}
	}

	// Send begin event
	ctx.sendEvent(&SyncEventBegin{
		LocalPath:  localPath,
		RemotePath: remotePath,
	})

	// Prepare end event
	success := false
	defer func() {
		ctx.sendEvent(&SyncEventEnd{
			LocalPath:  localPath,
			RemotePath: remotePath,
			IsUpload:   false,
			Success:    success,
		})
	}()

	// Create writer for local file
	localWriter, err := os.Create(localPath)
	if err != nil {
		ctx.sendEvent(&SyncEventError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Err:        err,
		})
		return
	}
	defer localWriter.Close()

	// Get reader for drive file
	remoteReader, err := ctx.t.DownloadDriveItem(item)
	if err != nil {
		ctx.sendEvent(&SyncEventError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Err:        err,
		})
		return
	}
	defer remoteReader.Close()

	// Write remote contents to local file
	if _, err = io.Copy(localWriter, remoteReader); err != nil {
		ctx.sendEvent(&SyncEventError{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Err:        err,
		})
		return
	}

	// Success!
	success = true
}

func (ctx *syncContext) syncFilesIdentical(local SyncFile, remote *DriveItem) bool {
	if local.Size != remote.Size {
		return false
	}

	// Hash local file
	hasher := quickxor.NewHasher()
	f, err := os.Open(local.FileName)
	if err != nil {
		return false
	}
	io.Copy(hasher, f)

	// Compare hashes
	return hasher.GetHashBase64() == remote.File.Hashes.QuickXor
}

// A read-only sync of a given OneDrive folder.
// Downloads files that don't exist in local directory.
// Deletes files in local directory not found on OneDrive.
// Does not redownload existing (up-to-date) files.
func (t *GraphToken) SyncFolder(remotePath string, localPath string, filterFn SyncFilterFn, eventFn SyncEventFn) error {
	// Create local output directory
	if err := os.MkdirAll(localPath, os.ModePerm); err != nil {
		return err
	}

	// List local directory
	localList, err := os.ReadDir(localPath)
	if err != nil {
		return err
	}
	localFiles := make(map[string]SyncFile)
	for _, v := range localList {
		size := int64(0)
		info, _ := v.Info()
		if info != nil && !info.IsDir() {
			size = info.Size()
		}
		localFiles[v.Name()] = SyncFile{
			FileName: filepath.Join(localPath, v.Name()),
			IsDir:    v.IsDir(),
			Size:     size,
		}
	}

	// Initialize sync job
	ctx := syncContext{
		remotePath: remotePath,
		localPath:  localPath,
		localFiles: localFiles,
		filterFn:   filterFn,
		eventFn:    eventFn,
		c:          make(chan *DriveItem, 32),
		t:          t,
	}
	for i := 0; i < 5; i++ {
		go ctx.syncQueue()
	}

	// Get OneDrive files
	onlineList, err := t.ListFolder(remotePath)
	if err != nil {
		return err
	}
	for _, item := range onlineList {
		syncFile := SyncFile{
			FileName: path.Join(remotePath, item.Name),
			IsDir:    item.IsDir(),
			Size:     int64(item.Size),
		}
		if filterFn != nil && filterFn(syncFile) {
			continue
		}

		// Do the thing
		ctx.addItem(item)
	}

	// Wait for downloads to complete
	ctx.wg.Wait()

	// Remove remaining files in folder
	for _, localFile := range localFiles {
		if filterFn != nil && filterFn(localFile) {
			continue
		}

		// Remove file
		err := os.Remove(localFile.FileName)
		if err != nil {
			ctx.sendEvent(&SyncEventError{
				LocalPath: localFile.FileName,
				Err:       err,
			})
		} else {
			ctx.sendEvent(SyncEventDelete{
				LocalPath: localFile.FileName,
			})
		}
	}

	// All good
	return nil
}
