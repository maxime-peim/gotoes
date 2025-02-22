package strava

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

const AddTimestampURL = GotoesURL + "/gotoes/strava/Add_Timestamps_To_GPX.php"

// Retrieve the pif value from the AddTimestampURL page.
// Act as a CSRF token.
func getPifValue() (string, error) {
	req, err := http.NewRequest("GET", AddTimestampURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create new request")
	}

	SetHeaders(req)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	docWelcomePage, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to load html page")
	}

	html, err := docWelcomePage.Html()
	if err != nil {
		return "", errors.Wrap(err, "failed to get html")
	}

	matches := PifRegex.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "", errors.New("failed to find pif value")
	}

	pif := matches[1]
	return pif, nil
}

// Prepare the request to add timestamps to a GPX file.
// This request is a multipart request, upload the GPX file and some parameters.
func postGpxForm(params *AddTimestampsToGPXParams) (string, error) {
	// Get the pif value from the AddTimestampURL page.
	pif, err := getPifValue()
	if err != nil {
		return "", errors.Wrap(err, "failed to get pif value")
	}

	body := new(bytes.Buffer)
	mwriter := multipart.NewWriter(body)

	w, err := mwriter.CreateFormFile("files[]", params.GPXFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to create form file")
	}

	gpx, err := os.Open(params.GPXFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to read gpx file")
	}
	defer gpx.Close()

	_, err = io.Copy(w, gpx)
	if err != nil {
		return "", errors.Wrap(err, "failed to copy gpx file")
	}

	mwriter.WriteField("timeZone", "Europe/Paris")
	mwriter.WriteField("spoofStartTime", params.StartTime)
	mwriter.WriteField("desiredSpeed", strconv.Itoa(int(params.DesiredSpeed)))
	mwriter.WriteField("MoK", "K") // K for Kilometers
	mwriter.WriteField("considerElevation", "")
	mwriter.WriteField("needsTimeStamp", "Y")
	mwriter.WriteField("convert_fit_files_to_csv", "")
	mwriter.WriteField("check_if_modified_by_gotoes", "")
	mwriter.WriteField("hasJava", "YES")
	mwriter.WriteField("pif", pif)

	if err = mwriter.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close mutliwriter")
	}

	reqUpload, err := http.NewRequest("POST", UploadURL, body)
	if err != nil {
		return "", errors.Wrap(err, "failed to create new request")
	}

	SetHeaders(reqUpload,
		NewHTTPHeader("Content-Type", mwriter.FormDataContentType()),
	)

	// Send the request.
	respUpload, err := httpClient.Do(reqUpload)
	if err != nil {
		return "", errors.Wrap(err, "failed to send request")
	}
	defer respUpload.Body.Close()

	// should be of the form {"count":"<a href=\"uploadProgress.php?f=8009945965089910\">...</a>"}
	respUploadRaw, err := io.ReadAll(respUpload.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	if respUpload.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected status code: %d\nContent: %s", respUpload.StatusCode, respUploadRaw)
	}

	respUploadContent := map[string]any{}
	if err = json.Unmarshal(respUploadRaw, &respUploadContent); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal response body")
	}

	if _, ok := respUploadContent["count"]; !ok {
		return "", errors.New("failed to find count in response")
	}

	msg := respUploadContent["count"].(string)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(msg))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse response body")
	}

	// Find the file id in the response.
	fileID := ""
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		matches := UploadRegex.FindStringSubmatch(href)
		if len(matches) < 2 {
			return
		}
		fileID = matches[1]
	})

	if fileID == "" {
		return "", errors.New("failed to find file id")
	}

	return fileID, nil
}

func getUploadForm(fileID string, params *AddTimestampsToGPXParams) (*url.Values, error) {
	// Prepare the request to download the GPX file.
	// Load the upload page with the file id and parse the form to get the default values.
	reqGetUpload, err := http.NewRequest("GET", UploadURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new request")
	}

	SetHeaders(reqGetUpload)

	q := reqGetUpload.URL.Query()
	q.Add("f", fileID)
	q.Add("timeZone", "Europe/Paris")
	q.Add("needsTimeStamp", "Y")
	q.Add("timeShift", "")
	q.Add("MoK", "K")
	q.Add("desiredSpeed", strconv.Itoa(int(params.DesiredSpeed)))
	q.Add("spoofStartTime", params.StartTime)
	q.Add("considerElevation", "bikespeed")
	q.Add("reverseRoute", "")
	reqGetUpload.URL.RawQuery = q.Encode()

	respGetUpload, err := httpClient.Do(reqGetUpload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}

	doc, err := goquery.NewDocumentFromReader(respGetUpload.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse response body")
	}

	values := &url.Values{}

	// default values from the form.
	doc.Find("form[name=combineParameters] input").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		value, _ := s.Attr("value")
		hidden, _ := s.Attr("type")
		if hidden != "hidden" {
			return
		}
		values.Add(name, value)
	})

	// adjust the values to download the GPX file.
	values.Add("outputFormat", "GPX")
	values.Add("timeZoneAdjustmentFactor", "3600")
	values.Add("ActivitySport", "Biking")

	return values, nil
}

func postUploadForm(values *url.Values) (io.ReadCloser, error) {
	body := strings.NewReader(values.Encode())
	reqPostUpload, err := http.NewRequest("POST", UploadURL, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new request")
	}

	SetHeaders(reqPostUpload,
		NewHTTPHeader("Content-Type", "application/x-www-form-urlencoded"),
	)

	// Download the GPX file. This should redirect to the file.
	respPostUpload, err := httpClient.Do(reqPostUpload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}

	if respPostUpload.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", respPostUpload.StatusCode)
	}

	return respPostUpload.Body, nil
}

func AddTimestampsToGPX(params *AddTimestampsToGPXParams) error {
	if params == nil {
		return errors.New("params is nil")
	}

	// Prepare the request to add timestamps to the GPX file.
	fileID, err := postGpxForm(params)
	if err != nil {
		return errors.Wrap(err, "failed to prepare add timestamp header")
	}

	// Get the form values to get the download link
	values, err := getUploadForm(fileID, params)
	if err != nil {
		return errors.Wrap(err, "failed to get form values")
	}

	// Post the form to download the GPX file.
	gpxReader, err := postUploadForm(values)
	if err != nil {
		return errors.Wrap(err, "failed to post upload form")
	}
	defer gpxReader.Close()

	outputFile := fmt.Sprintf("downloaded/GOTOES_%s.gpx", fileID)
	if params.OutputFile != "" {
		outputFile = params.OutputFile
	}

	// Save the downloaded file.
	downloadedFile, err := os.Create(outputFile)
	if err != nil {
		return errors.Wrap(err, "failed to open downloaded file")
	}
	defer downloadedFile.Close()

	_, err = io.Copy(downloadedFile, gpxReader)
	if err != nil {
		return errors.Wrap(err, "failed to copy downloaded file")
	}

	fmt.Println("Downloaded file:", downloadedFile.Name())
	return nil
}
