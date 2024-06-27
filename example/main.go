package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sukus21/gonedrive"
)

func main() {
	dat, _ := os.ReadFile("graphtoken.json")
	t := &gonedrive.GraphToken{}
	json.Unmarshal(dat, t)

	fmt.Println("Authenticating...")
	t, err := gonedrive.CreateAccess(
		"[app client ID goes here]",
		"http://127.0.0.1:8090/auth",
		t,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	//Sync files
	err = t.SyncFolder("Musik/mp3tag", "songs", Mp3Filter, EventHandler)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func Mp3Filter(file gonedrive.SyncFile) bool {
	if file.IsDir {
		return true
	}
	return strings.ToLower(filepath.Ext(file.FileName)) != ".mp3"
}

var mux sync.Mutex

func EventHandler(msg gonedrive.SyncMsg) {
	mux.Lock()
	defer mux.Unlock()

	fmt.Println("{\n\taction:", msg.Action, "\n\tmsg:", msg.Message, "\n\tlocal:", msg.LocalPath, "\n\tremote:", msg.RemotePath, "\n}")
}
