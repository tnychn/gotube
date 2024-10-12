package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/tnychn/gotube"
)

var (
	app         = kingpin.New("gotubedl", "A command-line YouTube video downloader powered by gotube.")
	idurl       = app.Arg("idurl", "Target video ID or video URL.").String()
	ls          = app.Flag("streams", "List all available streams of the video.").Short('s').Bool()
	lc          = app.Flag("captions", "List all available captions of the video.").Short('c').Bool()
	itag        = app.Flag("itag", "Download stream by the given itag.").Short('i').Uint()
	best        = app.Flag("best", "Download best stream of the given type. [a | v | av | a+v]").Short('b').String()
	lang        = app.Flag("lang", "Download caption with the given language code.").Short('l').String()
	destdir     = app.Flag("dest", "Destination output directory.").Short('d').ExistingDir()
	filename    = app.Flag("filename", "Destination video filename.").Short('f').String()
	noprefermp4 = app.Flag("no-prefer-mp4", "Toggle preference to mp4 formats.").Short('n').Bool()
	overwrite   = app.Flag("overwrite", "Overwrite existing file that has the same filename.").Short('o').Bool()
)

func printError(err error) {
	if e, is := err.(gotube.Error); is {
		color.Red("\r✘ %s: %v", e.Name(), err)
	} else {
		color.Red("\r✘ error: %v", err)
	}
}

func printVideo(video *gotube.Video) {
	printField := func(key string, value interface{}) {
		color.HiCyan("  %-9s %v", key+":", color.WhiteString("%v", value))
	}

	convertSecsToMinSecs := func(secs int) string {
		minutes := secs / 60
		seconds := secs % 60
		return fmt.Sprintf("%d:%d", minutes, seconds)
	}

	fmt.Printf("\r%s\n", strings.Repeat(" ", len("# Loading video...")))
	printField("Title", video.Title)
	printField("Channel", video.Author)
	printField("Duration", convertSecsToMinSecs(video.Duration))
	printField("Views", video.Views)
	printField("Age18+", video.IsAgeRestricted)
	printField("Unlisted", video.IsUnlisted)
	fmt.Println()
}

func main() {
	/* [Tests]
	normal: https://www.youtube.com/watch?v=vT3GUKuAzIs
	verified: https://www.youtube.com/watch?v=O6FXmdoGut8
	verified+18+: https://www.youtube.com/watch?v=07FYdnEawAQ
	caption: https://www.youtube.com/watch?v=aLJMEs_9ZZE
	live: https://www.youtube.com/watch?v=NEmIPDs9CZw
	*/

	app.HelpFlag.Short('h')
	app.Version("1.0.0")
	app.Author("tnychn")
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *idurl == "" {
		app.Fatalf("no idurl provided")
	}
	if *best != "" && (*best != "a" && *best != "v" && *best != "av" && *best != "a+v") {
		app.Fatalf("invalid -b/--best option: %v", *best)
	}

	_, _ = color.New(color.FgHiBlack).Print("# Loading Video...")
	video, err := gotube.NewVideo(*idurl, true)
	if err != nil {
		printError(err)
		return
	}
	printVideo(video)

	if *ls {
		listStreams(video.Streams())
	}
	if *lc {
		listCaptions(video.Captions())
	}
	if *itag != 0 || *best != "" {
		streams := video.Streams()
		var pendingStreams gotube.Streams
		// select stream
		if *itag != 0 {
			s := streams.Itag(int(*itag))
			if s != nil {
				pendingStreams = append(pendingStreams, s)
			}
		} else if *best != "" {
			if !*noprefermp4 {
				streams = streams.Filter(func(i int, stream gotube.Stream) bool {
					return stream.Subtype() == "mp4"
				})
			}
			switch *best {
			case "a":
				s := streams.Audios().Best()
				if s != nil {
					pendingStreams = append(pendingStreams, s)
				}
			case "v":
				s := streams.Videos().Best()
				if s != nil {
					pendingStreams = append(pendingStreams, s)
				}
			case "av":
				s := streams.Videos().WithAudio().Best()
				if s != nil {
					pendingStreams = append(pendingStreams, s)
				}
			case "a+v":
				s2 := streams.Videos().Best()
				s1 := streams.Filter(func(i int, stream gotube.Stream) bool {
					if *noprefermp4 {
						return true
					}
					return stream.Subtype() == s2.Subtype()
				}).Audios().Best()
				if s1 != nil && s2 != nil {
					pendingStreams = append(pendingStreams, s1, s2)
				}
			}
		}
		// download if not nil
		if len(pendingStreams) > 0 {
			if !*ls {
				listStreams(pendingStreams)
			}
			if len(pendingStreams) == 1 {
				_, _ = downloadStream(pendingStreams[0])
			}
			if len(pendingStreams) == 2 {
				downloadStreams(pendingStreams...)
			}
		} else {
			printError(fmt.Errorf("no matched stream"))
		}
	}
	if *lang != "" {
		captions := video.Captions()
		caption := captions.LanguageCode(*lang)
		if caption != nil {
			saveCaption(caption)
		} else {
			printError(fmt.Errorf("no caption with language code '%s' was found", *lang))
		}
	}
}

func listStreams(streams gotube.Streams) {
	printField := func(key string, value interface{}) {
		color.HiCyan("       %-11s %v", key+":", color.WhiteString("%v", value))
	}
	printVideoStream := func(stream *gotube.VideoStream) {
		printField("VCodec", stream.VideoCodec)
		if stream.HasAudio {
			printField("ACodec", stream.AudioCodec)
		}
		printField("Quality", stream.QualityLabel)
	}
	printAudioStream := func(stream *gotube.AudioStream) {
		printField("Codec", stream.Codec)
		printField("Quality", stream.Quality.String())
		printField("SampleRate", stream.SampleRate)
	}

	_, _ = color.New(color.FgWhite, color.Bold).Printf("List Streams:\n")
	if len(streams) == 0 {
		color.Yellow("  No Data")
	} else {
		for _, stream := range streams {
			color.Blue("  [%03d]%s", stream.Itag(), "------------------------------")
			metadata := stream.Metadata()
			printField("Type", metadata["mime_type"])
			switch s := stream.(type) {
			case *gotube.VideoStream:
				printVideoStream(s)
			case *gotube.AudioStream:
				printAudioStream(s)
			}
			printField("Bitrate", metadata["bitrate"])
			filesize := int64(metadata["file_size"].(float64))
			if filesize == 0 {
				printField("Filesize", "UNKNOWN")
			} else {
				printField("Filesize", fmt.Sprintf("%d MiB (%d Bytes)", filesize/1024/1024, filesize))
			}
		}
	}
	fmt.Println()
}

func listCaptions(captions gotube.Captions) {
	printField := func(key string, value interface{}) {
		color.HiCyan("      %s %v", key+":", color.WhiteString("%v", value))
	}
	_, _ = color.New(color.FgWhite, color.Bold).Printf("List Captions: %s\n", color.HiBlackString("# all available caption tracks"))
	if len(captions) == 0 {
		color.Yellow("  No Data")
	} else {
		for _, caption := range captions {
			color.Blue("  [%s]%s", caption.LanguageCode, "------------------------------")
			printField("Name", caption.Name)
		}
	}
	fmt.Println()
}

func downloadStream(stream gotube.Stream) (path string, err error) {
	filesize := int64(stream.Metadata()["file_size"].(float64))
	bar := progressbar.NewOptions64(
		filesize,
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionSetWidth(35),
	)
	fname := *filename
	if *best == "a+v" {
		fname = ""
	}
	if path, err = stream.Download(*destdir, fname, *overwrite, nil,
		func(written int64) {
			_ = bar.Set64(written)
		}); err != nil {
		printError(err)
	}
	bar.Finish()
	if err == nil {
		_, _ = color.New(color.FgGreen, color.Bold).Printf("# Download Finished %s\n", color.HiWhiteString(path))
	}
	return
}

func downloadStreams(streams ...gotube.Stream) {
	var paths []string
	for i, stream := range streams {
		task := "Audio"
		if i == 1 {
			task = "Video"
		}
		_, _ = color.New(color.FgHiBlack).Printf("# Downloading %s...\n", task)
		path, err := downloadStream(stream)
		if err != nil {
			return
		}
		paths = append(paths, path)
	}
	if finalpath, err := ffmpegRemux(paths, streams); err != nil {
		printError(err)
	} else {
		_, _ = color.New(color.FgYellow).Printf("\r# Done. Enjoy the video! %s\n", color.HiWhiteString(finalpath))
	}
}

func ffmpegRemux(paths []string, streams gotube.Streams) (finalpath string, err error) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return
	}
	fname := *filename
	if fname == "" {
		fname = streams[1].ParentVideo().ID
	}
	ext := streams[1].Subtype()
	if streams[0].Subtype() != streams[1].Subtype() {
		ext = "mkv"
	}
	finalpath = filepath.Join(*destdir, fname+"."+ext)
	_, _ = color.New(color.FgHiBlack).Print("# Remuxing...")
	cmd := exec.Command(bin, "-i", paths[0], "-i", paths[1], "-codec", "copy", finalpath)
	if err = cmd.Run(); err != nil {
		return
	}
	for _, path := range paths {
		if err = os.Remove(path); err != nil {
			return
		}
	}
	return
}

func saveCaption(caption *gotube.Caption) {
	var path string
	var err error
	_, _ = color.New(color.FgHiBlack).Print("# Saving caption...")
	if path, err = caption.Save(*destdir, "", true); err != nil {
		printError(err)
		return
	}
	_, _ = color.New(color.FgGreen, color.Bold).Printf("\r# Saved Caption %s\n", color.HiWhiteString(path))
}
