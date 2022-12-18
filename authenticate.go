package gonedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/browser"
)

const scope = "Files.Read Files.Read.All offline_access"

func CreateAccess(clientID string, redirectURI string, fname string) (*GraphToken, error) {
	gotToken := false
	t := &GraphToken{
		clientID:    clientID,
		redirectURI: redirectURI,
	}

	//Try using old token with refresh
	if fname != "" {
		f, err := os.ReadFile(fname)
		if err == nil {
			json.Unmarshal(f, t)
			err := t.refresh()
			if err == nil {
				gotToken = true
			}
		}
	}

	//Get new token
	if !gotToken {
		err := t.generateNew()
		if err != nil {
			return nil, err
		}
	}

	//Save token
	if fname != "" {
		b, _ := json.MarshalIndent(t, "", "\t")
		os.WriteFile(fname, b, 0666)
	}
	return t, nil
}

func (t *GraphToken) generateNew() error {
	responseType := "code"
	responseMode := "query"
	state := "0"
	err := browser.OpenURL(fmt.Sprintf(
		"https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize?client_id=%s&response_type=%s&redirect_uri=%s&response_mode=%s&scope=%s&state=%s",
		t.clientID,
		responseType,
		t.redirectURI,
		responseMode,
		scope,
		state,
	))
	if err != nil {
		return err
	}

	//Make first contant
	code := ""
	server := &http.Server{}
	server.Addr = ":8090"
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/auth",
		func(w http.ResponseWriter, req *http.Request) {

			msg := ""
			defer func() {
				err = nil
				if msg != "" {
					err = fmt.Errorf(msg)
				}
				w.Write([]byte("you can close this window now thanks :)"))
				server.Shutdown(context.TODO())
			}()

			//Verify state
			qs := req.URL.Query()
			rstate, ok := qs["state"]
			if !ok || len(rstate) != 1 || rstate[0] != state {
				if !ok {
					msg = "query does not contain 'state'"
				} else if len(rstate) != 1 {
					msg = "'state' query has more/less than 1 value"
				} else if rstate[0] != state {
					msg = "invalid 'state' value in query"
				}
				return
			}

			//Get access code
			rcode, ok := qs["code"]
			if !ok || len(rcode) != 1 {
				if !ok {
					msg = "query does not contain 'code'"
				} else if len(rstate) != 1 {
					msg = "'code' query has more/less than 1 value"
				}
				return
			}
			code = rcode[0]
		},
	)
	server.Handler = mux
	server.ListenAndServe()
	if err != nil {
		return err
	}

	//Request access token
	grant := "authorization_code"
	resp, err := http.Post(
		"https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(fmt.Sprintf(
			"client_id=%s&scope=%s&code=%s&redirect_uri=%s&grant_type=%s",
			t.clientID,
			scope,
			code,
			t.redirectURI,
			grant,
		)),
	)
	if err != nil {
		return err
	}

	//Handle access token
	body, _ := io.ReadAll(resp.Body)
	return json.Unmarshal(body, t)
}

func (t *GraphToken) refresh() error {
	grant := "refresh_token"
	resp, err := http.Post(
		"https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(fmt.Sprintf(
			"client_id=%s&scope=%s&refresh_token=%s&grant_type=%s",
			t.clientID,
			scope,
			t.RefreshToken,
			grant,
		)),
	)
	if err != nil {
		return err
	}

	//Handle access token
	body, _ := io.ReadAll(resp.Body)
	return json.Unmarshal(body, t)
}

func (t *GraphToken) MakeRequest(method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "https://graph.microsoft.com/v1.0"+url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+t.AccessToken)
	return http.DefaultClient.Do(req)
}
