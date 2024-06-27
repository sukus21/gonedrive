package gonedrive

import (
	"encoding/json"
	"io"
	"net/http"
)

// Builds a request object.
// Authorization is taken care of when the request is returned.
func (t *GraphToken) BuildRequestRaw(method string, uri string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "Bearer "+t.AccessToken)
	return request, nil
}

// Builds a request object.
// You supply the method, the endpoint (/me/drive/*) and the request body.
// Authorization is taken care of when the request is returned.
func (t *GraphToken) BuildRequest(method string, endpoint string, body io.Reader) (*http.Request, error) {
	return t.BuildRequestRaw(method, "https://graph.microsoft.com/v1.0"+endpoint, body)
}

// Sends a request object to the MS graph API.
// Returns error responses from the API as a go error.
func (t *GraphToken) SendRequest(request *http.Request) (*http.Response, error) {
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	// Error response?
	if response.StatusCode > 202 {
		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		err = &ErrorResponse{}
		json.Unmarshal(responseBody, err)
		return nil, err
	}

	// All good
	return response, nil
}

// Utility function for making requests.
// You supply the method, the endpoint (/me/drive/*) and the request body.
// Returns error responses from the API as a go error.
//
// Only the first value of contentType is used, the rest are ignored.
func (t *GraphToken) MakeRequest(method string, url string, body io.Reader, contentType ...string) (*http.Response, error) {
	req, err := t.BuildRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if len(contentType) != 0 {
		req.Header.Add("Content-Type", contentType[0])
	}
	return t.SendRequest(req)
}

func SendRequest[T any](t *GraphToken, request *http.Request) (*T, error) {
	response, err := t.SendRequest(request)
	if err != nil {
		return nil, err
	}

	// Read response body
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// Decode body
	dec := new(T)
	err = json.Unmarshal(responseBody, dec)
	return dec, err
}

// Utility function for making requests.
// You supply the method, the endpoint (/me/drive/*) and the request body.
// Returns error responses from the API as a go error.
// Deserializes the result to whatever you want, using json.Unmarshal.
func MakeRequest[T any](t *GraphToken, method string, url string, requestBody io.Reader) (*T, error) {
	request, err := t.BuildRequest(method, url, requestBody)
	if err != nil {
		return nil, err
	}

	return SendRequest[T](t, request)
}
