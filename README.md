<h1 align="center">gotube</h1>

<p align="center">
    <strong>Retrieve YouTube Videos in Golang</strong>
</p>

<p align="center">
    <a href="./go.mod"><img alt="go mod version" src="https://img.shields.io/github/go-mod/go-version/tnychn/gotube"></a>
    <a href="https://github.com/tnychn/gotube/releases"><img alt="release" src="https://img.shields.io/github/v/release/tnychn/gotube"></a>
    <a href="./LICENSE.txt"><img alt="license" src="https://img.shields.io/github/license/tnychn/gotube.svg"></a>
    <a href="https://pkg.go.dev/github.com/tnychn/gotube?tab=doc"><img alt="godoc" src="https://godoc.org/github.com/tnychn/gotube?status.svg"></a>
</p>

**gotube** is a lightweight yet reliable Go library (and command-line utility) for interacting with YouTube videos.
You can retrieve their information, streams and captions, as well as downloading them.

## Quickstart

```go
video, err := gotube.NewVideo("https://www.youtube.com/watch?v=9vc-I9rvGsw", true)
stream := video.Streams().Videos().Best() // <- obtain the highest quality video stream
path, err := stream.Download("./videos/", "", true,
    func(total int64) {
        fmt.Println("Total:", total)
    },
    func(written int64) {
        fmt.Print(written, "\r")
    },
)
// (errors are ignored for better readability)
```

## Features

* 🚸 Easy to use, fast and lightweight (CLI included)
* ✨ Minimalist-designed library interface
* 📞 Custom callbacks for downloading streams (`onStart` and `onProgress`)
* 🔍 Powerful stream querying methods
* 🎼 Support for both progressive and adaptive streams
* 💨 Fast downloading (parallel download with the file splitted into parts)
* 📑 Ability to extract detailed video information (including thumbnails)
* 📄 Support for retrieving video captions and save them in WebVTT format
* 🔞 ~~Support age-restricted videos~~
* 🔒 Support for encrypted videos
* 📦 Without external dependencies (except for the CLI)

## Installation

### Get Prebuilt Binaries

Please go to the [releases](https://github.com/tnychn/gotube/releases) page for downloads.

### Go Get

> Use this method if you wish to use **gotube** as a library as well as its command-line program.

Please make sure you have *Go 1.13+* installed in your machine.
```bash
$ go get -u github.com/tnychn/gotube/...
```

### Build From Source

Please make sure you have *Go 1.13+* installed in your machine.
```bash
$ git clone https://github.com/tnychn/gotube.git
$ cd gotube
$ go build cmd/gotubedl/main.go
# then run ./main to get started
```

## Usage

> The fastest way to get started is to learn by examples!

Don't forget to **import** the package first!
```go
import "github.com/tnychn/gotube"
```

Let's begin by **getting the video**!
```go
// You can either use the url of the video ...
video, err := gotube.NewVideo("https://www.youtube.com/watch?v=9vc-I9rvGsw", true)
// or you can simply use the video id ...
video, err := gotube.NewVideo("9vc-I9rvGsw", true)
```

If the second parameter (`preinit`) is set to `false`, you will need to call `Initialize()` afterwards
before accessing any fields and methods.

> For more information, visit the [documentations](https://pkg.go.dev/github.com/tnychn/gotube?tab=doc).

### Obtaining Streams

```go
streams := video.Streams() // --> Streams ([]Stream)
```

#### Selecting a single stream

These methods return a single `Stream`.

```go
streams.First() // select the first stream (index=0)
streams.Last() // select the last stream (index=-1)
streams.Itag(238) // select by itag
```

You may need to do type assertion afterwards to handle different types of streams (`VideoStream` vs `AudioStream`).

Since `Stream` is an interface, you cannot access the fields of the
implemented struct (either `VideoStream` or `AudioStream`) directly. Therefore, here is where `Stream.Metadata()` comes into place.

`Stream.Metadata()` marshals the fields of the stream itself into a `map[interface{}]` (the keys are marshaled according to the fields' JSON tags) so that you can access the fields of the stream.
However you must be very careful while using this method, as there are fields that only exists in `VideoStream` but not `AudioStream`, vice versa.

#### Sorting streams

The `Streams.Sort()` method accepts a `less` parameter, which is a `func(Stream, Stream) bool`.

It is almost identical with the `less` functions of Go's `sort` package.

```go
// Sort streams by their filesize in ascending order
streams = streams.Sort(func (stream1, stream2 Stream) bool {
    return stream1.Metadata()["file_size"].(int64) < stream2.Metadata()["file_size"].(int64)
})
```

#### Filtering streams

```go
// only keep .mp4 format video streams
streams = streams.Filter(func (stream Stream) bool {
    return stream.Subtype() == "mp4"
})
// ...or you can simply use this shorthand method
streams = streams.Subtype("mp4")
```

```go
// To get video streams only,
vstreams := streams.Videos() // --> VideoStreams

// To get video streams that also has audio,
avstreams := streams.Videos().WithAudio() // --> VideoStreams

// To get audio streams only,
astreams := streams.Audios() // --> AudioStreams
```

**Selecting the best stream**

```go
// To get highest resolution video stream,
streams.Videos().Best() // --> *VideoStream or nil

// To get highest resolution video stream that also has audio,
streams.Videos().WithAudio().Best() // --> *VideoStream or nil

// To get highest bitrate audio stream,
streams.Audios().Best() // --> *AudioStream or nil
```

**Downloading stream**

```go
path, err := stream.Download("../music", "favourite_song", true,
    func(total int64) {
        fmt.Println("Total", total)
    },
    func(written int64) {
        fmt.Print(written, "\r")
    },
) // path: /Users/tony/music/favourite_song.mp4
```

### Obtaining Captions

```go
captions := video.Captions() // --> Captions ([]*Caption)
```

> `Caption` is a struct, not an interface

**Selecting by language code**

```go
caption := captions.LanguageCode("en")
```

This is currently the only available method of the `Captions` type.

**Saving to disk**

```go
path, err := caption.Save("../captions", "english", true)
fmt.Println(path) // path: /Users/tony/captions/english.vtt
```

### Handling Errors

Check if the error returned implements the `gotube.Error` interface.

```go
if e, is := err.(gotube.Error); is {
    fmt.Println(e.Name())
}
```

## Command-line Interface

```text
usage: gotubedl [<flags>] [<idurl>]

A command-line YouTube video downloader powered by gotube.

Flags:
  -h, --help               Show context-sensitive help.
  -s, --streams            List all available streams of the video.
  -c, --captions           List all available captions of the video.
  -i, --itag=ITAG          Download stream by the given itag.
  -b, --best=BEST          Download best quality stream of the given type. [a | v | av | a+v]
  -l, --lang=LANG          Download caption with the given language code.
  -d, --dest=DEST          Destination output directory.
  -f, --filename=FILENAME  Destination video filename.
  -n, --no-prefer-mp4      Toggle preference to mp4 formats.
  -o, --overwrite          Overwrite existing file that has the same filename.
      --version            Show application version.

Args:
  [<idurl>]  Target video ID or video URL.
```

**Download the best audio stream**

```bash
$ gotubedl "https://www.youtube.com/watch?v=9vc-I9rvGsw" -b a
```

**Download the best video stream**

```bash
$ gotubedl "https://www.youtube.com/watch?v=9vc-I9rvGsw" -b v
```

**Download the best video stream (with audio)**

```bash
$ gotubedl "https://www.youtube.com/watch?v=9vc-I9rvGsw" -b av
```

**Download the best stream (remuxing)**

When using `a+v`, both the best audio stream and the best video stream will be downloaded.
Then, `gotubedl` will execute `ffmpeg` to combine the audio and the video into a single video file.

If `no-prefer-mp4` is **not** specified, the audio stream and video stream chosen will be in `mp4` formats,
and the final output video file will result a `mp4` as well.

Otherwise, the audio stream and video stream chosen may be in different formats
and the final output video file will result in a `mkv` in this case.

```bash
$ gotubedl "https://www.youtube.com/watch?v=9vc-I9rvGsw" -b a+v
```

## TODOs

- [ ] Add support for playlists
- [ ] Fix support for age-restricted videos

## Credits

This project is inspired by [@nficano](https://github.com/nficano)'s [pytube](https://github.com/nficano/pytube).

---

<div align="center">
    <sub><strong>~ crafted with ♥︎ by tnychn ~</strong></sub>
    <br>
    <sub><strong>MIT © 2020 Tony Chan</strong></sub>
</div>
