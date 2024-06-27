package gonedrive

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/sukus21/gonedrive/quickxor"
)

type SyncAction string

type SyncMsg struct {
	Action     SyncAction
	Message    string
	LocalPath  string
	RemotePath string
}

type SyncFile struct {
	// Full file path
	FileName string

	// File size, not relevant for directories
	Size int64

	// Is this a directory?
	IsDir bool
}

type SyncFilter func(file SyncFile) bool
type SyncEvent func(msg SyncMsg)

func IgnoreFolders(file SyncFile) bool {
	return file.IsDir
}

type syncContext struct {
	t          *GraphToken
	mux        sync.Mutex
	wg         sync.WaitGroup
	c          chan *DriveItem
	localFiles map[string]SyncFile
	filterFn   SyncFilter
	eventFn    SyncEvent
	remotePath string
	localPath  string
}

func (ctx *syncContext) addItem(item *DriveItem) {
	ctx.wg.Add(1)
	ctx.c <- item
}

func (ctx *syncContext) syncQueue() {
	for item := range ctx.c {
		msg := ctx.HandleItem(item)
		if ctx.eventFn != nil {
			ctx.eventFn(msg)
		}
	}
}

func (ctx *syncContext) HandleItem(item *DriveItem) SyncMsg {
	defer ctx.wg.Done()

	// Find local file in map
	ctx.mux.Lock()
	localFile, exists := ctx.localFiles[item.Name]
	delete(ctx.localFiles, item.Name)
	ctx.mux.Unlock()

	// I'll be using these
	remotePath := path.Join(ctx.remotePath, item.Name)
	localPath := filepath.Join(ctx.localPath, item.Name)
	buildMsg := func(action SyncAction, message string) SyncMsg {
		return SyncMsg{
			Action:     action,
			Message:    message,
			RemotePath: remotePath,
			LocalPath:  localPath,
		}
	}

	// Cannot sync directories at the moment
	if item.IsDir() {
		return buildMsg("error", "remote file is directory")
	}

	// Handle existing local file
	if exists {
		// Cannot sync directories
		if localFile.IsDir {
			return buildMsg("error", "local file is directory")
		}

		// Is local file identical?
		if ctx.syncFilesIdentical(localFile, item) {
			return buildMsg("skip", "up to date")
		}

		// No they are not, remove local file
		if err := os.Remove(localPath); err != nil {
			return buildMsg("error", err.Error())
		}
	}

	// Download new file from remote
	remoteReader, err := ctx.t.DownloadDriveItem(item)
	if err != nil {
		return buildMsg("error", err.Error())
	}

	//Save file and all that
	localWriter, err := os.Create(localPath)
	if err != nil {
		return buildMsg("error", err.Error())
	}
	defer localWriter.Close()
	if _, err = io.Copy(localWriter, remoteReader); err != nil {
		return buildMsg("error", err.Error())
	}

	// Success!
	return buildMsg("downloaded", "downloaded")
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
func (t *GraphToken) SyncFolder(remotePath string, localPath string, filterFn SyncFilter, eventFn SyncEvent) error {
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
	fmt.Println("Getting list of files in folder...")
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

		// Remove file >:)
		err := os.Remove(localFile.FileName)
		if eventFn != nil {
			if err != nil {
				eventFn(SyncMsg{
					Action:    "error",
					Message:   "could not delete local file",
					LocalPath: localFile.FileName,
				})
			} else {
				eventFn(SyncMsg{
					Action:    "delete",
					Message:   "removed excess local file",
					LocalPath: localFile.FileName,
				})
			}
		}
	}

	// All good
	return nil
}
