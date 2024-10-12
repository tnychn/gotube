package gotube

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tnychn/gotube/data"
	"github.com/tnychn/gotube/decrypt"
	"github.com/tnychn/gotube/errors"
	"github.com/tnychn/gotube/extract"
	"github.com/tnychn/gotube/utils"
)

// VideoInfo carries information of a video.
type VideoInfo struct {
	Title           string
	Description     string
	Keywords        []string
	Category        string
	Duration        int
	Views           int
	AverageRating   float64
	Author          string
	ThumbnailURL    string
	IsAgeRestricted bool
	IsUnlisted      bool
}

// Video represents a YouTube video object, carrying the information, streams and captions of the video.
type Video struct {
	*VideoInfo

	ID              string
	WatchURL        string
	EmbedURL        string
	IsAgeRestricted bool

	streams  Streams
	captions Captions

	watchHTML string
	embedHTML string

	infoQuery url.Values
	infoRaw   string
	infoURL   string

	js    string
	jsURL string

	decryption     *decrypt.Decryption
	playerResponse *data.PlayerResponse
}

// NewVideo returns a new video object of the given `idurl` (can be one of video id or video url).
// If `preinit` is true, all necessary HTTP requests will be done and data will be fetched on initialization.
// Otherwise, you will need to call `Initialize()` yourself before accessing any fields of this video object.
func NewVideo(idurl string, preinit bool) (vid *Video, err error) {
	id, err := extract.VideoID(idurl)
	if err != nil {
		return nil, err
	}

	vid = new(Video)
	vid.ID = id
	vid.WatchURL = fmt.Sprintf("https://youtube.com/watch?hl=en&v=%v", id)
	vid.EmbedURL = fmt.Sprintf("https://youtube.com/embed/%v", id)
	vid.streams = nil
	vid.captions = nil
	if preinit {
		err = vid.Initialize()
	}
	return
}

// Initialize performs all necessary HTTP requests and descrambles the fetched data.
func (video *Video) Initialize() (err error) {
	if err = video.prefetch(); err != nil {
		return
	}
	if err = video.descramble(); err != nil {
		return
	}
	if err = video.obtainBasicInfo(); err != nil {
		return
	}
	return nil
}

func (video *Video) prefetch() (err error) {
	// Watch HTML
	content, err := utils.HttpFetch(video.WatchURL)
	if err != nil {
		return
	}
	video.watchHTML = string(content)

	video.IsAgeRestricted = extract.AgeRestricted(video.watchHTML)
	// Embed HTML (if video is age-restricted)
	if video.IsAgeRestricted {
		if content, err = utils.HttpFetch(video.EmbedURL); err != nil {
			return
		}
		video.embedHTML = string(content)
	}

	// Raw Info (in a form of url queries)
	fetchInfo := func(el string) error {
		if video.IsAgeRestricted {
			sts := extract.Sts(video.embedHTML)
			video.infoURL = utils.MakeAgeRestrictedInfoURL(video.ID, sts)
		} else {
			video.infoURL = utils.MakeInfoURL(video.ID, el)
		}
		if content, err := utils.HttpFetch(video.infoURL); err != nil {
			return err
		} else {
			video.infoRaw = string(content)
		}
		return nil
	}
	// check and fallback
	err = fetchInfo("embedded")
	if err != nil {
		if _, is := err.(errors.HttpError); is {
			return fetchInfo("detailpage")
		}
		return err
	}
	values, err := url.ParseQuery(video.infoRaw)
	if err != nil {
		return err
	}
	if regexp.MustCompile(`UNPLAYABLE`).MatchString(values.Get("player_response")) {
		return fetchInfo("detailpage")
	}
	return nil
}

func (video *Video) descramble() (err error) {
	// Descramble `PlayerResponse` from the url-query-format response of the 'get_video_info' endpoint
	values, err := url.ParseQuery(video.infoRaw)
	if err != nil {
		return
	}
	video.infoQuery = values
	// - handle errors
	if video.infoQuery.Get("status") != "ok" {
		return errors.RequestFailedError{Reason: video.infoQuery.Get("reason")}
	}
	// - parse player response
	resp := video.infoQuery.Get("player_response")
	if resp == "" {
		return fmt.Errorf("empty 'player_response'")
	}
	if err = json.Unmarshal([]byte(resp), &video.playerResponse); err != nil {
		return
	}
	if video.playerResponse.PlayabilityStatus.Status != "OK" {
		return errors.RequestFailedError{Reason: video.playerResponse.PlayabilityStatus.Status} // video.playerResponse.PlayabilityStatus.Reason
	}

	// Descramble `PlayerConfig` in order to find the endpoint to 'base.js' for later use (i.e. stream decryption)
	html := video.watchHTML
	if video.IsAgeRestricted {
		html = video.embedHTML
	}
	config, err := extract.PlayerConfig(html)
	if err != nil {
		return
	}
	playerConfig := struct {
		Args struct {
			EmbeddedPlayerResponse string `json:"embedded_player_response"`
		} `json:"args"`
		Assets struct {
			CSS string `json:"css"`
			JS  string `json:"js"`
		} `json:"assets"`
	}{}
	if err = json.Unmarshal([]byte(config), &playerConfig); err != nil {
		return
	}
	video.jsURL = "https://youtube.com" + playerConfig.Assets.JS
	return nil
}

func (video *Video) obtainBasicInfo() (err error) {
	details := video.playerResponse.VideoDetails
	if details.IsLiveContent {
		return errors.VideoUnsupportedError{VideoID: video.ID}
	}
	video.VideoInfo = new(VideoInfo)
	video.Title = details.Title
	video.Description = details.ShortDescription
	video.Keywords = details.Keywords
	video.Category = video.playerResponse.Microformat.PlayerMicroformatRenderer.Category
	video.Duration, _ = strconv.Atoi(details.LengthSeconds)
	video.Views, _ = strconv.Atoi(details.ViewCount)
	video.AverageRating = details.AverageRating
	video.Author = details.Author
	if len(details.Thumbnail.Thumbnails) > 0 {
		video.ThumbnailURL = details.Thumbnail.Thumbnails[len(details.Thumbnail.Thumbnails)-1].URL
	} else {
		video.ThumbnailURL = fmt.Sprintf("https://img.youtube.com/vi/%v/maxresdefault.jpg", video.ID)
	}
	video.IsUnlisted = video.playerResponse.Microformat.PlayerMicroformatRenderer.IsUnlisted
	return nil
}

func (video *Video) stream(format data.StreamFormat) Stream {
	filesize, _ := strconv.ParseInt(format.ContentLength, 10, 64)
	expirationSecs, _ := strconv.ParseInt(video.playerResponse.StreamingData.ExpiresInSeconds, 10, 64)
	expiration := time.Now().Add(time.Second * time.Duration(expirationSecs))
	mime, codecs := extract.MimeCodecs(format.MimeType)
	if strings.Split(mime, "/")[0] == "audio" {
		sampleRate, _ := strconv.ParseInt(format.AudioSampleRate, 10, 64)
		return &AudioStream{
			video:          video,
			itag:           format.Itag,
			cipher:         format.Cipher,
			rawurl:         format.URL,
			FileSize:       filesize,
			MimeType:       mime,
			Codec:          codecs[0],
			Quality:        format.AudioQuality,
			Bitrate:        int64(format.Bitrate),
			AverageBitrate: int64(format.AverageBitrate),
			SampleRate:     sampleRate,
			Channels:       format.AudioChannels,
			Expiration:     expiration,
		}
	}
	quality := format.QualityLabel
	if quality == "" {
		quality = format.Quality
	}
	var acodec string
	hasAudio := len(codecs)%2 == 0
	if hasAudio {
		acodec = codecs[1]
	}
	return &VideoStream{
		video:          video,
		itag:           format.Itag,
		cipher:         format.Cipher,
		rawurl:         format.URL,
		FileSize:       filesize,
		MimeType:       mime,
		VideoCodec:     codecs[0],
		AudioCodec:     acodec,
		Bitrate:        int64(format.Bitrate),
		AverageBitrate: int64(format.AverageBitrate),
		QualityLabel:   quality,
		Width:          format.Width,
		Height:         format.Height,
		HasAudio:       hasAudio,
		Expiration:     expiration,
	}
}

// Streams retrieves all the available streams that belong to this video.
func (video *Video) Streams() Streams {
	if video.playerResponse == nil {
		panic("player response is nil: Initialize() must be called beforehand")
	}
	if video.streams != nil {
		return video.streams
	}
	formats := append(video.playerResponse.StreamingData.Formats, video.playerResponse.StreamingData.AdaptiveFormats...)
	for _, format := range formats {
		video.streams = append(video.streams, video.stream(format))
	}
	video.streams.Sort(func(stream1 Stream, stream2 Stream) bool {
		return stream1.Itag() < stream2.Itag()
	})
	return video.streams
}

func (video *Video) caption(track data.CaptionTrack) *Caption {
	return &Caption{
		URL:          track.BaseURL,
		Name:         track.Name.SimpleText,
		LanguageCode: track.LanguageCode,
	}
}

// Captions retrieves all the available captions that belong to this video.
func (video *Video) Captions() Captions {
	if video.playerResponse == nil {
		panic("player response is nil: Initialize() must be called beforehand")
	}
	if video.captions != nil {
		return video.captions
	}
	for _, track := range video.playerResponse.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks {
		video.captions = append(video.captions, video.caption(track))
	}
	return video.captions
}
