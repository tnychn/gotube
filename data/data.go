package data

import "encoding/xml"

// const GAPIkey string = "AIzaSyCIM4EzNqi1in22f4Z3Ru3iYvLaY8tc3bo"

type AudioQuality string

const (
	AudioQualityLow    AudioQuality = "AUDIO_QUALITY_LOW"
	AudioQualityMedium AudioQuality = "AUDIO_QUALITY_MEDIUM"
	AudioQualityHigh   AudioQuality = "AUDIO_QUALITY_HIGH"
)

func (aq AudioQuality) String() (s string) {
	switch aq {
	case AudioQualityLow:
		s = "low"
	case AudioQualityMedium:
		s = "medium"
	case AudioQualityHigh:
		s = "high"
	}
	return
}

type StreamFormat struct {
	Itag             int          `json:"itag"`
	Cipher           string       `json:"cipher"`
	URL              string       `json:"url"`
	MimeType         string       `json:"mimeType"`
	Bitrate          int          `json:"bitrate"`
	Width            int          `json:"width"`
	Height           int          `json:"height"`
	LastModified     string       `json:"lastModified"`
	ContentLength    string       `json:"contentLength"`
	Quality          string       `json:"quality"`
	QualityLabel     string       `json:"qualityLabel"`
	ProjectionType   string       `json:"projectionType"`
	AverageBitrate   int          `json:"averageBitrate"`
	ApproxDurationMs string       `json:"approxDurationMs"`
	AudioSampleRate  string       `json:"audioSampleRate"`
	AudioChannels    int          `json:"audioChannels"`
	AudioQuality     AudioQuality `json:"audioQuality"`
}

type VideoDetails struct {
	VideoID          string   `json:"videoId"`
	Title            string   `json:"title"`
	LengthSeconds    string   `json:"lengthSeconds"`
	Keywords         []string `json:"keywords"`
	ChannelID        string   `json:"channelId"`
	IsOwnerViewing   bool     `json:"isOwnerViewing"`
	ShortDescription string   `json:"shortDescription"`
	IsCrawlable      bool     `json:"isCrawlable"`
	Thumbnail        struct {
		Thumbnails []struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"thumbnails"`
	} `json:"thumbnail"`
	AverageRating     float64 `json:"averageRating"`
	AllowRatings      bool    `json:"allowRatings"`
	ViewCount         string  `json:"viewCount"`
	Author            string  `json:"author"`
	IsPrivate         bool    `json:"isPrivate"`
	IsUnpluggedCorpus bool    `json:"isUnpluggedCorpus"`
	IsLiveContent     bool    `json:"isLiveContent"`
}

type CaptionTrack struct {
	BaseURL string `json:"baseUrl"`
	Name    struct {
		SimpleText string `json:"simpleText"`
	} `json:"name"`
	VssID          string `json:"vssId"`
	LanguageCode   string `json:"languageCode"`
	Kind           string `json:"kind"`
	IsTranslatable bool   `json:"isTranslatable"`
}

type PlayerResponse struct {
	PlayabilityStatus struct {
		Status          string `json:"status"`
		Reason          string `json:"reason"`
		PlayableInEmbed bool   `json:"playableInEmbed"`
		ContextParams   string `json:"contextParams"`
	} `json:"playabilityStatus"`
	StreamingData struct {
		Formats          []StreamFormat `json:"formats"`
		AdaptiveFormats  []StreamFormat `json:"adaptiveFormats"`
		ExpiresInSeconds string         `json:"expiresInSeconds"`
	} `json:"streamingData"`
	Captions struct {
		PlayerCaptionsRenderer struct {
			BaseURL    string `json:"baseUrl"`
			Visibility string `json:"visibility"`
		} `json:"playerCaptionsRenderer"`
		PlayerCaptionsTracklistRenderer struct {
			CaptionTracks []CaptionTrack `json:"captionTracks"`
			AudioTracks   []struct {
				CaptionTrackIndices []int `json:"captionTrackIndices"`
			} `json:"audioTracks"`
			TranslationLanguages []struct {
				LanguageCode string `json:"languageCode"`
				LanguageName struct {
					SimpleText string `json:"simpleText"`
				} `json:"languageName"`
			} `json:"translationLanguages"`
			DefaultAudioTrackIndex int `json:"defaultAudioTrackIndex"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
	VideoDetails VideoDetails `json:"videoDetails"`
	Microformat  struct {
		PlayerMicroformatRenderer struct {
			Embed struct {
				IframeURL      string `json:"iframeUrl"`
				FlashURL       string `json:"flashUrl"`
				Width          int    `json:"width"`
				Height         int    `json:"height"`
				FlashSecureURL string `json:"flashSecureUrl"`
			} `json:"embed"`
			IsUnlisted       bool   `json:"isUnlisted"`
			Category         string `json:"category"`
			PublishDate      string `json:"publishDate"`
			OwnerChannelName string `json:"ownerChannelName"`
			UploadDate       string `json:"uploadDate"`
		} `json:"playerMicroformatRenderer"`
	} `json:"microformat"`
}

type Transcript struct {
	XMLName xml.Name         `xml:"transcript"`
	Texts   []TranscriptText `xml:"text"`
}

type TranscriptText struct {
	Start    float64 `xml:"start,attr"`
	Duration float64 `xml:"dur,attr"`
	Text     string  `xml:",chardata"`
}
