package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/tnychn/gotube/errors"
	"github.com/tnychn/gotube/utils"
)

type Writer struct {
	lock       sync.Mutex
	written    int64
	onProgress func(written int64)
}

func (w *Writer) Write(d []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	n := len(d)
	w.written += int64(n)
	if w.onProgress != nil {
		w.onProgress(w.written)
	}
	return n, nil
}

func Download(dlurl string, destdir, filename, ext string, overwrite bool, onStart func(total int64), onProgress func(written int64)) (finalpath string, err error) {
	if destdir == "" {
		if destdir, err = os.Getwd(); err != nil {
			return
		}
	}
	if destdir, err = filepath.Abs(destdir); err != nil {
		return
	}
	finalpath = filepath.Join(destdir, filename+"."+ext)
	// FIXME
	if _, err = os.Stat(finalpath); !overwrite && os.IsExist(err) {
		return
	}

	// Send HEAD request to get file content length
	header, err := utils.HttpHead(dlurl)
	if err != nil {
		return
	}
	clen := header.Get("Content-Length")
	var total int64
	if clen != "" {
		total, _ = strconv.ParseInt(clen, 10, 64)
	}
	if onStart != nil {
		onStart(total)
	}
	workerNum := int64(10)
	chunk := total / workerNum

	var errs []error
	var paths []string
	writer := &Writer{onProgress: onProgress}
	group := new(sync.WaitGroup)
	for i := int64(0); i < workerNum; i++ {
		min := chunk * i
		max := chunk*(i+1) - 1
		if i == workerNum-1 {
			max = total
		}

		path := filepath.Join(destdir, fmt.Sprintf("%s.%d.part", filename, i+1))
		paths = append(paths, path)
		group.Add(1)
		go func(i, min, max int64, path string) {
			defer group.Done()
			h := make(http.Header)
			h.Add("Range", fmt.Sprintf("bytes=%d-%d", min, max))
			resp, err := utils.HttpRequest(http.MethodGet, dlurl, &h)
			if err != nil {
				errs = append(errs, err)
				return
			}
			if resp.StatusCode >= 300 {
				errs = append(errs, errors.HttpError{StatusCode: resp.StatusCode})
				return
			}
			defer resp.Body.Close()

			file, err := os.Create(path)
			if err != nil {
				errs = append(errs, err)
				return
			}
			defer file.Close()
			if _, err = io.Copy(file, io.TeeReader(resp.Body, writer)); err != nil {
				return
			}
		}(i, min, max, path)
	}
	group.Wait()
	if len(errs) > 0 {
		err = errs[0]
		return
	}
	err = mergeFiles(finalpath, paths)
	return
}

func mergeFiles(finalpath string, paths []string) (err error) {
	var files []*os.File
	defer func() {
		for _, file := range files {
			file.Close()
			_ = os.Remove(file.Name())
		}
	}()
	file, err := os.Create(finalpath)
	if err != nil {
		return
	}
	for _, path := range paths {
		f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		files = append(files, f)
		if _, err = io.Copy(file, f); err != nil {
			return err
		}
	}
	return file.Close()
}
