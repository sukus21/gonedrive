package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sukus21/gonedrive"
)

//go:embed clientid.txt
var clientId string

func main() {
	dat, _ := os.ReadFile("graphtoken.json")
	t := &gonedrive.GraphToken{}
	json.Unmarshal(dat, t)

	fmt.Println("Authenticating...")
	t, err := gonedrive.CreateAccess(
		clientId,
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

// Prints events as they happen
func EventHandler(msg gonedrive.SyncEvent) {
	mux.Lock()
	defer mux.Unlock()
	fmt.Println(msg)
}
