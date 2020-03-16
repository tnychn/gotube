package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/tnychn/gotube/errors"
)

const UserAgent string = "Mozilla/5.0"

func HttpRequest(method, u string, headers *http.Header) (*http.Response, error) {
	request, err := http.NewRequest(method, u, nil)
	if err != nil {
		panic(err)
	}
	if headers != nil {
		request.Header = *headers
	}
	request.Header.Set("User-Agent", UserAgent)
	client := http.Client{Timeout: time.Second * 120}
	return client.Do(request)
}

func HttpHead(u string) (header http.Header, err error) {
	response, err := HttpRequest(http.MethodGet, u, nil)
	if err != nil {
		return
	}
	if response.StatusCode >= 300 {
		return nil, errors.HttpError{StatusCode: response.StatusCode}
	}
	return response.Header, nil
}

func HttpFetch(u string) ([]byte, error) {
	response, err := HttpRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		return nil, errors.HttpError{StatusCode: response.StatusCode}
	}
	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func MakeInfoURL(videoID, el string) string {
	eurl := url.PathEscape("https://youtube.googleapis.com/v/" + videoID)
	u, _ := url.ParseRequestURI("https://www.youtube.com/get_video_info")
	query := u.Query()
	query.Add("video_id", videoID)
	query.Add("hl", "en")
	query.Add("el", el)
	query.Add("eurl", eurl)
	u.RawQuery = query.Encode()
	return u.String()
}

func MakeAgeRestrictedInfoURL(videoID, sts string) string {
	eurl := url.PathEscape("https://youtube.googleapis.com/v/" + videoID)
	u, _ := url.ParseRequestURI("https://www.youtube.com/get_video_info")
	query := u.Query()
	query.Add("video_id", videoID)
	query.Add("hl", "en")
	query.Add("el", "detailpage")
	query.Add("eurl", eurl)
	query.Add("sts", sts)
	u.RawQuery = query.Encode()
	return u.String()
}

func StructToMap(s interface{}) map[string]interface{} {
	d, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	m := make(map[string]interface{})
	if err = json.Unmarshal(d, &m); err != nil {
		panic(err)
	}
	return m
}
