package gonedrive

import "fmt"

type GraphToken struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	clientID    string
	redirectURI string
}

type DriveItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Size         int    `json:"size"`
	WebURL       string `json:"webUrl"`
	Ctag         string `json:"cTag"`
	Etag         string `json:"eTag"`
	CreationDate string `json:"createdDateTime"`
	ModifiedDate string `json:"lastModifiedDateTime"`

	Root        *struct{} `json:"root"`
	DownloadURL string    `json:"@content.downloadUrl"`

	Audio *struct {
		Album             string `json:"album"`
		AlbumArtist       string `json:"albumArtist"`
		Artist            string `json:"Artist"`
		Bitrate           int    `json:"bitrate"`
		Composers         string `json:"composers"`
		Copyright         string `json:"copyright"`
		Disc              int    `json:"disc"`
		DiscCount         int    `json:"discCount"`
		Duration          int    `json:"duration"`
		Genre             string `json:"genre"`
		HasDRM            bool   `json:"hasDrm"`
		IsVariableBitrate bool   `json:"isVariableBitrate"`
		Title             string `json:"title"`
		Track             int    `json:"track"`
		TrackCount        int    `json:"trackCount"`
		Year              int    `json:"year"`
	} `json:"audio"`

	File *struct {
		MimeType string  `json:"mimeType"`
		Hashes   *Hashes `json:"hashes"`
	} `json:"file"`

	Folder *struct {
		ChildCount int `json:"childCount"`
		View       *struct {
			SortBy    string `json:"sortBy"`
			SortOrder string `json:"sortOrder"`
			ViewType  string `json:"viewType"`
		} `json:"view"`
	} `json:"folder"`
}

type Hashes struct {
	Crc32    string `json:"crc32Hash"`
	Sha1     string `json:"sha1Hash"`
	Sha256   string `json:"sha256Hash"`
	QuickXor string `json:"quickXorHash"`
}

type ErrorResponse struct {
	Outer struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Inner   struct {
			Date            string `json:"date"`
			RequestID       string `json:"request-id"`
			ClientRequestId string `json:"client-request-id"`
		} `json:"innerError"`
	} `json:"error"`
}

func (err *ErrorResponse) Error() string {
	return fmt.Sprintf(
		"%s: %s\n%s, %s",
		err.Outer.Code,
		err.Outer.Message,
		err.Outer.Inner.Date,
		err.Outer.Inner.RequestID,
	)
}

type ChildrenResponse struct {
	Context  string       `json:"@odata.context"`
	Count    int          `json:"@odata.count"`
	NextLink string       `json:"@odata.nextLink"`
	Data     []*DriveItem `json:"value"`
}
