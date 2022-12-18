package main

import (
	"fmt"

	"github.com/sukus21/gonedrive"
)

func main() {
	fmt.Println("Authenticating...")
	t, err := gonedrive.CreateAccess(
		"[app client ID goes here]",
		"http://127.0.0.1:8090/auth",
		"graphauth.json",
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
