package gonedrive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"time"
)

func (t *GraphToken) UploadContent(r io.Reader, size int64, destPath string, params UploadSessionParams) (*DriveItem, error) {
	// Get upload session
	session, err := t.CreateUploadSession(destPath, params)
	if err != nil {
		return nil, err
	}
	uploadUrl := session.UploadURL

	// Upload file, 1MB at a time
	buf := make([]byte, 0x100000)
	pos := int64(0)
	for {
		n, err := io.ReadFull(r, buf[:min(int64(len(buf)), size-pos)])
		isEof := pos+int64(n) == size
		if err != nil && (!isEof && (err != io.EOF || err != io.ErrUnexpectedEOF)) {
			return nil, err
		}

		// Create request
		r := bytes.NewReader(buf[:n])
		request, err := http.NewRequest("PUT", uploadUrl, r)
		if err != nil {
			return nil, err
		}

		// Write content headers
		request.Header.Add("Content-Length", fmt.Sprint(n))
		request.Header.Add("Content-Range", fmt.Sprintf(
			"bytes %d-%d/%d",
			pos,
			pos+int64(n)-1,
			size,
		))
		pos += int64(n)

		// Send
		if !isEof {
			_, err := t.SendRequest(request)
			if err != nil {
				return nil, err
			}
		} else {
			return SendRequest[DriveItem](t, request)
		}
	}
}

func (t *GraphToken) UploadFile(file fs.File, destPath string, conflictBehaviour ConflictBehaviour) (*DriveItem, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Build parameters
	modTime := stat.ModTime()
	params := UploadSessionParams{
		ConflictBehaviour: ConflictBehaviour_Fail,
		ModifiedAt:        &modTime,
	}

	totalSize := stat.Size()
	return t.UploadContent(file, totalSize, destPath, params)
}

type UploadSessionParams struct {
	ConflictBehaviour ConflictBehaviour
	CreatedAt         *time.Time
	AccessedAt        *time.Time
	ModifiedAt        *time.Time
}

func (p *UploadSessionParams) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"@odata.type":                       "microsoft.graph.driveItemUploadableProperties",
		"@microsoft.graph.conflictBehavior": p.ConflictBehaviour,
	}

	if p.CreatedAt != nil || p.AccessedAt != nil || p.ModifiedAt != nil {
		fileInfo := map[string]string{}
		if p.CreatedAt != nil {
			fileInfo["createdDateTime"] = p.CreatedAt.Format(time.RFC3339)
		}
		if p.AccessedAt != nil {
			fileInfo["lastAccessedDateTime"] = p.AccessedAt.Format(time.RFC3339)
		}
		if p.ModifiedAt != nil {
			fileInfo["lastModifiedDateTime"] = p.ModifiedAt.Format(time.RFC3339)
		}
		out["fileSystemInfo"] = fileInfo
	}

	// Final output
	return json.Marshal(map[string]any{"item": out})
}

func (t *GraphToken) CreateUploadSession(destPath string, params UploadSessionParams) (*UploadSessionResponse, error) {
	// Create request body
	requestData, _ := params.MarshalJSON()
	requestBody := bytes.NewReader(requestData)

	// Create upload session
	urlPath := EndpointPath(destPath, "createUploadSession")
	response, err := t.MakeRequest("POST", "/me/drive/"+urlPath, requestBody, "application/json")
	if err != nil {
		return nil, err
	}

	// Read response body
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal and return
	dec := new(UploadSessionResponse)
	err = json.Unmarshal(responseBody, dec)
	return dec, err
}
