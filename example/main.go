package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	err = t.SyncFolder("Musik/mp3tag", "songs", "audio/mpeg")
	if err != nil {
		fmt.Println(err)
		return
	}
}
