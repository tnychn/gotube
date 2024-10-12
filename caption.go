package gotube

import (
	"encoding/xml"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/tnychn/gotube/data"
	"github.com/tnychn/gotube/utils"
)

// Captions represents a sequence of captions.
type Captions []*Caption

// LanguageCode returns the `Caption` of the given language code.
func (captions Captions) LanguageCode(lc string) *Caption {
	for _, caption := range captions {
		if caption.LanguageCode == lc {
			return caption
		}
	}
	return nil
}

// Caption represents a caption of a YouTube video.
type Caption struct {
	content string

	URL          string
	Name         string
	LanguageCode string
}

// GetContent retrieves the content of this caption (most likely in xml format).
func (caption *Caption) GetContent() (string, error) {
	if caption.content == "" {
		content, err := utils.HttpFetch(caption.URL)
		if err != nil {
			return "", err
		}
		caption.content = string(content)
	}
	return caption.content, nil
}

// GetWebVTT first retrieves the content of this caption by calling `GetContent()` then converts and returns it in WebVTT format.
func (caption *Caption) GetWebVTT() (string, error) {
	timestr := func(f float64) string {
		i, frac := math.Modf(f)
		mins := int(i / 60)
		secs := int(math.Mod(i, 60))
		ms := fmt.Sprintf("%.3f", frac)
		return fmt.Sprintf("%02d:%d.%v", mins, secs, strings.ReplaceAll(ms, "0.", ""))
	}
	content, err := caption.GetContent()
	if err != nil {
		return "", err
	}
	// do conversion (xml to webvtt)
	var transcript data.Transcript
	if err = xml.Unmarshal([]byte(content), &transcript); err != nil {
		return "", err
	}
	lines := []string{"WEBVTT\n"}
	for _, text := range transcript.Texts {
		start := text.Start
		end := text.Start + text.Duration
		line := fmt.Sprintf("%v --> %v\n%v\n", timestr(start), timestr(end), html.UnescapeString(text.Text))
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

// Save downloads the content of this caption and saves it to a file in the local machine.
// If `destdir` is empty, it defaults to the current directory.
// If `filename` is empty, it defaults to the name of this caption.
// If `webvtt` is true, it saves the content in a WebVTT (.vtt) format file.
func (caption *Caption) Save(destdir, filename string, webvtt bool) (finalpath string, err error) {
	if destdir == "" {
		if destdir, err = os.Getwd(); err != nil {
			return
		}
	}
	if destdir, err = filepath.Abs(destdir); err != nil {
		return
	}
	if filename == "" {
		filename = caption.Name
	}

	var content string
	if webvtt {
		content, err = caption.GetWebVTT()
	} else {
		content, err = caption.GetContent()
	}
	if err != nil {
		return
	}

	path := filepath.Join(destdir, filename+".part")
	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	if _, err = file.WriteString(content); err != nil {
		return
	}

	ext := ".xml"
	if webvtt {
		ext = ".vtt"
	}
	finalpath = filepath.Join(destdir, filename+ext)
	err = os.Rename(path, finalpath)
	return
}
