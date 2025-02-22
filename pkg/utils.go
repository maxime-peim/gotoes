package strava

import (
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"time"
)

const (
	GotoesURL         = "https://gotoes.org"
	UploadURL         = GotoesURL + "/gotoes/strava/upload.php"
	UploadProgressURL = GotoesURL + "/gotoes/strava/uploadProgress.php"
	UserAgent         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

var (
	PifRegex    = regexp.MustCompile(`https:\/\/gotoes.org\/strava\/uploadtool.php\?pif=([0-9a-z]{32})`)
	UploadRegex = regexp.MustCompile(`uploadProgress.php\?f=([0-9]*)`)
	httpClient  = &http.Client{
		Timeout: 10 * time.Second,
	}
)

func init() {
	// Create a new cookie jar to store cookies.
	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar
}

type AddTimestampsToGPXParams struct {
	GPXFile      string
	OutputFile   string
	DesiredSpeed uint32
	StartTime    string
}

type HTTPHeader struct {
	Key   string
	Value string
}

func NewHTTPHeader(key, value string) HTTPHeader {
	return HTTPHeader{
		Key:   key,
		Value: value,
	}
}

func SetHeaders(req *http.Request, headers ...HTTPHeader) {
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", GotoesURL)
	req.Header.Set("Referer", AddTimestampURL)

	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
}
