package gonedrive

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const hashfile = "!._hashes.json"

// Helper function, creates URL from path and query.
func generatePath(path string, query []string) string {
	o := "root:/" + path + ":/children"
	if path == "" {
		o = "root/children"
	}
	for i, v := range query {
		if i == 0 {
			o += "?"
		} else {
			o += "&"
		}
		o += v
	}
	return o
}

// Get DriveItem information for a single file.
// Path should be WITHOUT leading/trailing slashes.
func (t *GraphToken) ListFile(path string) (*DriveItem, error) {
	urlpath := generatePath(path, nil)
	resp, err := t.MakeRequest("GET", "/me/drive/"+urlpath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//Read response body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//Invalid response?
	if resp.StatusCode > 202 {
		err = &ErrorResponse{}
		json.Unmarshal(body, err)
		return nil, err
	}

	//Decode body
	dec := &DriveItem{}
	err = json.Unmarshal(body, dec)
	return dec, err
}

// Lists all files in a given folder.
// Path should be WITHOUT leading/trailing slashes.
func (t *GraphToken) ListFolder(path string) ([]*DriveItem, error) {
	songlist := make([]*DriveItem, 0, 1024)
	query := []string{}

	for {
		resp, err := t.getChildren(path, query)
		if err != nil {
			return nil, err
		}
		songlist = append(songlist, resp.Data...)
		if resp.NextLink == "" {
			return songlist, nil
		}
		split := strings.SplitN(resp.NextLink, "?", 2)
		query = split[1:]
	}
}

// Runs a query to get all DriveItems within a folder.
func (t *GraphToken) getChildren(path string, query []string) (*ChildrenResponse, error) {

	//Generate proper path
	up := generatePath(path, query)

	//Send request to OneDrive
	resp, err := t.MakeRequest("GET", "/me/drive/"+up, nil)
	if err != nil {
		return nil, err
	}

	//Read response body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//Invalid response?
	if resp.StatusCode > 202 {
		err = &ErrorResponse{}
		json.Unmarshal(body, err)
		return nil, err
	}

	//Decode body
	dec := &ChildrenResponse{}
	err = json.Unmarshal(body, dec)
	return dec, err
}

// Downloads a DriveItem, and returns the file body.
func (t *GraphToken) DownloadFile(file *DriveItem) ([]byte, error) {
	resp, err := t.MakeRequest(
		"GET",
		fmt.Sprintf("/me/drive/items/%s/content", file.Id),
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// A read-only sync of a given OneDrive folder.
// Downloads files that don't exist in local directory.
// Deletes files in local directory not found on OneDrive.
// Does not redownload existing files.
// Creates a json file in the target directory to store state of synced files.
func (t *GraphToken) SyncFolder(odpath string, destpath string, filetype string) error {
	skipCount := 0
	downloadCount := 0
	errorCount := 0
	deletedCount := 0

	//Get OneDrive files
	fmt.Println("Getting list of files in folder...")
	files, err := t.ListFolder(odpath)
	if err != nil {
		return err
	}

	//Load previous hashes
	hash := make(map[string]string)
	unseen := make(map[string]int)
	b, err := os.ReadFile(destpath + "/" + hashfile)
	if err == nil {
		json.Unmarshal(b, &hash)
		fs, _ := os.ReadDir(destpath)
		for _, v := range fs {
			if v.Name() != hashfile {
				unseen[v.Name()] = 0
			}
		}
		for k := range hash {
			unseen[k] = 0
		}
	} else if _, err = os.Stat(destpath); err != nil {

		//Prepare destination folder
		r := strings.SplitAfter(destpath, "/")
		p := ""
		for i := 0; i < len(r); i++ {
			p += r[i]
			if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
				os.Mkdir(p, os.ModePerm)
			}
		}
	}

	//Mark files for deletion
	fmt.Println("Starting sync...")
	download := make([]*DriveItem, 0, len(files))

	//Maybe download files
	for _, v := range files {
		if v.File == nil {
			continue
		}
		if v.File.MimeType == filetype || filetype == "" {
			_, err := os.Stat(destpath + "/" + v.Name)
			if h, ok := hash[v.Name]; ok && h == v.File.Hashes.QuickXor && err == nil {

				//File exists is up to date
				fmt.Println("skipped: " + v.Name)
				skipCount++
			} else {

				//Mark file for download
				download = append(download, v)
			}
			delete(unseen, v.Name)
		}
	}

	//Delete unseen files
	for k := range unseen {
		delete(hash, k)
		path := destpath + "/" + k
		if _, err := os.Stat(path); err == nil {
			os.Remove(path)
			fmt.Println("deleted: " + k)
			deletedCount++
		}
	}

	//Prepare download
	jobs := make(chan *DriveItem, len(download))
	results := make(chan string, len(download))

	//Start downloading files
	hashmu := &sync.Mutex{}
	countmu := &sync.Mutex{}
	for i := 0; i < 5; i++ {
		go func() {
			for i := range jobs {

				//Download file
				b, err := t.DownloadFile(i)
				if err != nil {
					results <- "error: " + err.Error()
					countmu.Lock()
					errorCount++
					countmu.Unlock()
					continue
				}

				//Save file and all that
				path := destpath + "/" + i.Name
				err = os.WriteFile(path, b, os.ModePerm)
				if err != nil {
					results <- "error: " + err.Error()
					countmu.Lock()
					errorCount++
					countmu.Unlock()
					continue
				}

				//Write to hash file
				hashmu.Lock()
				hash[i.Name] = i.File.Hashes.QuickXor
				hashmu.Unlock()

				//Send result down result channel
				results <- "downloaded: " + i.Name
				countmu.Lock()
				downloadCount++
				countmu.Unlock()
			}
		}()
	}

	//Distribute work
	for _, v := range download {
		jobs <- v
	}
	close(jobs)

	//Show results
	for i := 0; i < len(download); i++ {
		res := <-results
		fmt.Println(res)
	}
	close(results)

	//Save hash file
	b, _ = json.MarshalIndent(hash, "", "\t")
	os.WriteFile(destpath+"/"+hashfile, b, os.ModePerm)

	//Print result of operation
	fmt.Printf(
		"downloaded: %d\nskipped: %d\ndeleted: %d\nerrors: %d\n",
		downloadCount,
		skipCount,
		deletedCount,
		errorCount,
	)
	return nil
}
