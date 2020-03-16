package gotube

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/tnychn/gotube/data"
	"github.com/tnychn/gotube/decrypt"
	"github.com/tnychn/gotube/download"
	"github.com/tnychn/gotube/utils"
)

func getDownloadURL(stream Stream) (string, error) {
	video := stream.ParentVideo()
	metadata := stream.Metadata()
	cipher := metadata["cipher"].(string)
	rawurl := metadata["rawurl"].(string)

	// stream url is already unencrypted
	if cipher == "" {
		return rawurl, nil
	}

	// check whether the stream url is encrypted or not
	query, err := url.ParseQuery(cipher)
	if err != nil {
		return "", err
	}
	u := query.Get("url")
	s := query.Get("s")
	isEncrypted := strings.HasPrefix(query.Get("sp"), "sig") ||
		(s != "" && !(strings.Contains(u, "&sig=") || strings.Contains(u, "&lsig=")))

	// decrypt encrypted stream url
	if isEncrypted {
		if video.js == "" {
			content, err := utils.HttpFetch(video.jsURL)
			if err != nil {
				return "", err
			}
			video.js = string(content)
		}
		if video.decryption == nil {
			video.decryption, err = decrypt.NewDecryption(video.js)
			if err != nil {
				return "", err
			}
		}
		signature, err := video.decryption.DecryptSignature(s)
		if err != nil {
			return "", err
		}
		if !strings.Contains(u, "&ratebypass=") {
			u += "&ratebypass=yes"
		}
		return u + "&sig=" + signature, nil
	}
	// stream url is (probably?) already unencrypted
	return u, nil
}

// VideoStreams represents a sequence of streams of type 'video' (can be with / without audio).
type VideoStreams []*VideoStream

// WithAudio returns a copy of `VideoStreams` containing only `VideoStream` objects that also has audio.
func (streams VideoStreams) WithAudio() (videoStreams VideoStreams) {
	for _, s := range streams {
		if s.HasAudio {
			videoStreams = append(videoStreams, s)
		}
	}
	return
}

// First returns the first `VideoStream` object in this `VideoStreams`.
func (streams VideoStreams) First() *VideoStream {
	if len(streams) == 0 {
		return nil
	}
	return streams[0]
}

// Last returns the last `VideoStream` object in this `VideoStreams`.
func (streams VideoStreams) Last() *VideoStream {
	if len(streams) == 0 {
		return nil
	}
	return streams[len(streams)-1]
}

// Best returns the `VideoStream` object with the highest resolution.
func (streams VideoStreams) Best() *VideoStream { return streams.sortByResolution().Last() }

// Worst returns the `VideoStream` object with the lowest resolution.
func (streams VideoStreams) Worst() *VideoStream { return streams.sortByResolution().First() }

func (streams VideoStreams) sortByResolution() VideoStreams {
	sort.Slice(streams, func(i, j int) bool {
		return streams[i].Height <= streams[j].Height && streams[i].Bitrate < streams[j].Bitrate
	})
	return streams
}

// AudioStreams represents a sequence of streams of type 'audio'.
type AudioStreams []*AudioStream

// First returns the first `AudioStream` object in this `AudioStreams`.
func (streams AudioStreams) First() *AudioStream {
	if len(streams) == 0 {
		return nil
	}
	return streams[0]
}

// Last returns the last `AudioStream` object in this `AudioStreams`.
func (streams AudioStreams) Last() *AudioStream {
	if len(streams) == 0 {
		return nil
	}
	return streams[len(streams)-1]
}

// Best returns the `AudioStream` object with the highest overall bitrate.
func (streams AudioStreams) Best() *AudioStream { return streams.sortByBitrate().Last() }

// Worst returns the `AudioStream` object with the lowest overall bitrate.
func (streams AudioStreams) Worst() *AudioStream { return streams.sortByBitrate().First() }

func (streams AudioStreams) sortByBitrate() AudioStreams {
	sort.Slice(streams, func(i, j int) bool {
		return streams[i].AverageBitrate < streams[j].AverageBitrate && streams[i].Bitrate < streams[j].Bitrate
	})
	return streams
}

// Streams represents a generic sequence of streams which can be of type 'video' or type 'audio'.
type Streams []Stream

// First returns the first `Stream` object in this `Streams`.
func (streams Streams) First() Stream {
	if len(streams) == 0 {
		return nil
	}
	return streams[0]
}

// Last returns the last `Stream` object in this `Streams`.
func (streams Streams) Last() Stream {
	if len(streams) == 0 {
		return nil
	}
	return streams[len(streams)-1]
}

// Itag returns the `Stream` of the given itag.
func (streams Streams) Itag(itag int) Stream {
	return streams.Filter(func(_ int, stream Stream) bool {
		return stream.Itag() == itag
	}).First()
}

// Subtype returns a copy of this `Streams` of the given subtype.
func (streams Streams) Subtype(subtype string) Streams {
	return streams.Filter(func(_ int, stream Stream) bool {
		return stream.Subtype() == subtype
	})
}

// Audios returns a `AudioStreams` (i.e. streams of type 'audio' only) from this `Streams`.
func (streams Streams) Audios() (audioStreams AudioStreams) {
	for _, s := range streams {
		if _, isAduio := s.(*AudioStream); isAduio {
			audioStreams = append(audioStreams, s.(*AudioStream))
		}
	}
	return
}

// Videos returns a `VideoStreams` (i.e. streams of type 'video' only) from this `Streams`.
func (streams Streams) Videos() (videoStreams VideoStreams) {
	for _, s := range streams {
		if _, isVideo := s.(*VideoStream); isVideo {
			videoStreams = append(videoStreams, s.(*VideoStream))
		}
	}
	return
}

// Filter returns a filtered copy of this `Streams` according to the conditions of `f`.
// If `f` returns true, keep the stream, remove otherwise.
func (streams Streams) Filter(f func(int, Stream) bool) (results Streams) {
	for i, stream := range streams {
		if f(i, stream) {
			results = append(results, stream)
		}
	}
	return results
}

// Sort returns a sorted copy of this `Streams` according to the conditions of `less` (works the same with the sort package).
func (streams Streams) Sort(less func(Stream, Stream) bool) (results Streams) {
	sort.Slice(streams, func(i, j int) bool {
		return less(streams[i], streams[j])
	})
	return results
}

// Stream is an interface which is implemented by `VideoStream` and `AudioStream`.
type Stream interface {
	ParentVideo() *Video
	Type() string
	Subtype() string
	Name() string
	Itag() int
	Metadata() map[string]interface{}
	GetDownloadURL() (string, error)
	Download(destdir, filename string, overwrite bool, onStart func(int64), onProgress func(int64)) (string, error)
}

// VideoStream represents a stream of type 'video'.
type VideoStream struct {
	video *Video

	itag   int
	cipher string
	rawurl string

	FileSize       int64     `json:"file_size"`
	MimeType       string    `json:"mime_type"`
	VideoCodec     string    `json:"video_codec"`
	AudioCodec     string    `json:"audio_codec"`
	Bitrate        int64     `json:"bitrate"`
	AverageBitrate int64     `json:"average_bitrate"`
	QualityLabel   string    `json:"quality_label"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	HasAudio       bool      `json:"has_audio"`
	Expiration     time.Time `json:"expiration"`
}

// ParentVideo returns a pointer to the video object that this stream belongs to.
func (stream *VideoStream) ParentVideo() *Video { return stream.video }

// Type returns the type of this stream. (video or video+audio)
func (stream *VideoStream) Type() string {
	if stream.HasAudio {
		return "video+audio"
	}
	return "video"
}

// Subtype returns the subtype (extension) gotten from mime type of this stream.
func (stream *VideoStream) Subtype() string {
	return strings.Split(stream.MimeType, "/")[1]
}

// Name returns the name of this stream.
func (stream *VideoStream) Name() string {
	return fmt.Sprintf("%s_%s_%s", stream.ParentVideo().ID, stream.QualityLabel, stream.Type())
}

// Itag returns the itag of this stream.
func (stream *VideoStream) Itag() int { return stream.itag }

// Metadata returns a map consisting all the fields of this stream.
func (stream *VideoStream) Metadata() map[string]interface{} {
	m := utils.StructToMap(*stream)
	m["rawurl"] = stream.rawurl
	m["cipher"] = stream.cipher
	return m
}

// GetDownloadURL gets the decrypted URL to the direct source of this stream that can be used for downloading.
func (stream *VideoStream) GetDownloadURL() (string, error) {
	return getDownloadURL(stream)
}

// Download downloads the content of this stream and saves it to a file in the local machine.
// If `destdir` is empty, it defaults to the current directory.
// If `filename` is empty, it defaults to the name of this stream (i.e. `Name()`).
// If `overwrite` is true, it will overwrite the existing file of the same filename, skip downloading otherwise.
// `onProgress` is called whenever `Write()` occurrs, with the total amount of written bytes provided as parameter.
func (stream *VideoStream) Download(destdir, filename string, overwrite bool, onStart func(total int64), onProgress func(written int64)) (path string, err error) {
	dlurl, err := getDownloadURL(stream)
	if err != nil {
		return
	}
	if filename == "" {
		filename = stream.Name()
	}
	return download.Download(dlurl, destdir, filename, stream.Subtype(), overwrite, onStart, onProgress)
}

// AudioStream represents a stream of type 'audio'.
type AudioStream struct {
	video *Video

	itag   int
	cipher string
	rawurl string

	FileSize       int64             `json:"file_size"`
	MimeType       string            `json:"mime_type"`
	Codec          string            `json:"codec"`
	Quality        data.AudioQuality `json:"quality"`
	Bitrate        int64             `json:"bitrate"`
	AverageBitrate int64             `json:"average_bitrate"`
	SampleRate     int64             `json:"sample_rate"`
	Channels       int               `json:"channels"`
	Expiration     time.Time         `json:"expiration"`
}

// ParentVideo returns a pointer to the video object that this stream belongs to.
func (stream *AudioStream) ParentVideo() *Video { return stream.video }

// Type returns the type of this stream. (audio)
func (stream *AudioStream) Type() string {
	return "audio"
}

// Subtype returns the subtype (extension) gotten from mime type of this stream.
func (stream *AudioStream) Subtype() string {
	return strings.Split(stream.MimeType, "/")[1]
}

// Name returns the name of this stream.
func (stream *AudioStream) Name() string {
	return fmt.Sprintf("%s_%s_%s", stream.ParentVideo().ID, stream.Quality.String(), stream.Type())
}

// Itag returns the itag of this stream.
func (stream *AudioStream) Itag() int { return stream.itag }

// Metadata returns a map consisting all the fields of this stream.
func (stream *AudioStream) Metadata() map[string]interface{} {
	m := utils.StructToMap(*stream)
	m["itag"] = stream.itag
	m["rawurl"] = stream.rawurl
	m["cipher"] = stream.cipher
	return m
}

// GetDownloadURL gets the decrypted URL to the direct source of this stream that can be used for downloading.
func (stream *AudioStream) GetDownloadURL() (string, error) {
	return getDownloadURL(stream)
}

// Download downloads the content of this stream and saves it to a file in the local machine.
// If `destdir` is empty, it defaults to the current directory.
// If `filename` is empty, it defaults to the name of this stream (i.e. `Name()`).
// If `overwrite` is true, it will overwrite the existing file of the same filename, skip downloading otherwise.
// `onProgress` is called whenever `Write()` occurrs, with the total amount of written bytes provided as parameter.
func (stream *AudioStream) Download(destdir, filename string, overwrite bool, onStart func(total int64), onProgress func(written int64)) (path string, err error) {
	dlurl, err := getDownloadURL(stream)
	if err != nil {
		return
	}
	if filename == "" {
		filename = stream.Name()
	}
	return download.Download(dlurl, destdir, filename, stream.Subtype(), overwrite, onStart, onProgress)
}
