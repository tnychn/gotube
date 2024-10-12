package extract

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/tnychn/gotube/errors"
)

func AgeRestricted(watchHTML string) bool {
	return regexp.MustCompile(`og:restrictions:age`).MatchString(watchHTML)
}

func VideoID(text string) (string, error) {
	re := regexp.MustCompile(`([0-9A-Za-z_-]{11})$`)
	URL, err := url.ParseRequestURI(text)
	if err != nil {
		if re.MatchString(text) {
			return text, nil
		}
	} else {
		if strings.Contains(URL.Hostname(), "youtube.com") {
			v := URL.Query().Get("v")
			if v != "" && re.MatchString(v) {
				return v, nil
			}
		} else if strings.Contains(URL.Hostname(), "youtu.be") {
			path := strings.TrimPrefix(URL.Path, "/")
			if path != "" && re.MatchString(path) {
				return path, nil
			}
		}
	}
	return "", errors.ExtractError{Caller: "video id", Pattern: re.String()}
}

func Sts(embedHTML string) string {
	matches := regexp.MustCompile(`sts"\s*:\s*(\d+)`).FindStringSubmatch(embedHTML)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

func MimeCodecs(mime string) (string, []string) {
	matches := regexp.MustCompile(`(\w+/\w+);\scodecs="([a-zA-Z-0-9.,\s]*)"`).FindStringSubmatch(mime)
	if len(matches) == 0 {
		return "", nil
	}
	mimeType := matches[1]
	codecs := strings.Split(matches[2], ",")
	for i, c := range codecs {
		codecs[i] = strings.TrimSpace(c)
	}
	return mimeType, codecs
}

func PlayerConfig(watchHTML string) (string, error) {
	patterns := []string{
		`;ytplayer\.config\s*=\s*({.*?});`,
		`;ytplayer\.config\s*=\s*({.+?});ytplayer`,
		`;yt\.setConfig\(\{'PLAYER_CONFIG':\s*({.*})}\);`,
		`;yt\.setConfig\(\{'PLAYER_CONFIG':\s*({.*})(,'EXPERIMENT_FLAGS'|;)`,
	}
	for _, pattern := range patterns {
		matches := regexp.MustCompile(pattern).FindStringSubmatch(watchHTML)
		if len(matches) == 0 {
			continue
		}
		return matches[1], nil
	}
	return "", errors.ExtractError{Caller: "player config", Pattern: "<player config patterns>"}
}
