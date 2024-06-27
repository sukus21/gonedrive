package gonedrive

import (
	"fmt"
	"io"
	"strings"
)

// Get information about a single drive item
func (t *GraphToken) GetDriveItem(path string, query ...string) (*DriveItem, error) {
	urlpath := EndpointPath(path, "")
	return MakeRequest[DriveItem](t, "GET", "/me/drive/"+urlpath, nil)
}

// Runs a query to get all DriveItems within a folder.
// This returns the paginated response structure.
// Path should be WITHOUT leading/trailing slashes.
func (t *GraphToken) GetDriveItemChildren(path string, query []string) (*ResponsePaginated[[]*DriveItem], error) {
	up := EndpointPath(path, "children", query...)
	return MakeRequest[ResponsePaginated[[]*DriveItem]](t, "GET", "/me/drive/"+up, nil)
}

// Lists all files in a given folder.
// Path should be WITHOUT leading/trailing slashes.
func (t *GraphToken) ListFolder(path string) ([]*DriveItem, error) {
	songlist := make([]*DriveItem, 0, 1024)
	query := []string{}

	for {
		resp, err := t.GetDriveItemChildren(path, query)
		if err != nil {
			return nil, err
		}
		songlist = append(songlist, resp.Value...)
		if resp.NextLink == "" {
			return songlist, nil
		}
		split := strings.SplitN(resp.NextLink, "?", 2)
		query = split[1:]
	}
}

// Downloads a DriveItem, and returns the file body.
func (t *GraphToken) DownloadDriveItem(item *DriveItem) ([]byte, error) {
	resp, err := t.MakeRequest("GET", fmt.Sprintf("/me/drive/items/%s/content", item.Id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
