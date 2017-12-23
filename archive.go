package main

import (
	"archive/zip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/spkg/zipfs"
)

// Archive holds information about single clio archive
type Archive struct {
	http.Handler

	UserName   string `json:"id"`
	ScreenName string `json:"name"`
	Type       string `json:"type"`

	filePath   string
	fileSystem *zipfs.FileSystem
	lk         sync.Mutex
	cleanTimer *time.Timer
}

func (a *Archive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.lk.Lock()
	if a.Handler == nil {
		var err error
		a.fileSystem, err = zipfs.New(a.filePath)
		if err != nil {
			a.fileSystem = nil
			a.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Cannot open archive for serve", http.StatusInternalServerError)
			})
		} else {
			a.Handler = http.FileServer(a.fileSystem)
		}
	}
	a.cleanTimer.Reset(time.Minute)
	a.lk.Unlock()
	a.Handler.ServeHTTP(w, r)
}

func (a *Archive) cleanFS() {
	a.lk.Lock()
	defer a.lk.Unlock()

	if a.fileSystem != nil {
		a.fileSystem.Close()
	}
	a.fileSystem = nil
	a.Handler = nil

}

// NewArchive reads zip file and fills Archive
func NewArchive(filePath string) (*Archive, error) {
	a := new(Archive)
	a.filePath = filePath
	zipReader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()

	var feedInfo *zip.File
	for _, zf := range zipReader.File {
		if strings.HasSuffix(zf.Name, "/_json/data/feedinfo.js") {
			feedInfo = zf
			break
		}
	}
	if feedInfo == nil {
		return nil, errors.New("cannot find 'feedinfo.js' file")
	}

	if err := readZipObject(feedInfo, a); err != nil {
		return nil, errors.Annotate(err, "cannot read 'feedinfo.js'")
	}

	a.cleanTimer = time.AfterFunc(0, a.cleanFS)

	return a, nil
}

func readZipObject(file *zip.File, v interface{}) error {
	r, err := file.Open()
	if err != nil {
		return errors.Annotate(err, "cannot open archived file")
	}
	defer r.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Annotate(err, "cannot read archived file")
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		return errors.Annotate(err, "cannot parse JSON")
	}

	return nil
}
