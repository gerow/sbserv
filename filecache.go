package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"time"
)

type FileCache struct {
	Path        string
	RecvChannel chan fcTask
	Closed      bool
	FileRefs    []FileRef
}

type SearchResult struct {
	FileRefs []FileRef
}

type fcTask struct {
	action string // actions can be refresh or search
	args   []interface{}
}

func (fc *FileCache) daemon() {
	for {
		task, ok := <-fc.RecvChannel
		if !ok {
			// the channel has been closed. just exit
			return
		}

		switch {
		case task.action == "refresh":
			fc.doRefresh()
		case task.action == "search":
			regex := task.args[0].(string)
			responseChannel := task.args[1].(chan []FileRef)
			fc.doSearch(regex, responseChannel)
		}
	}
}

func (fc *FileCache) doRefreshDirectory(realPath string, webPath string) {
	file, err := os.Open(realPath)
	defer file.Close()
	if err != nil {
		log.Println(err.Error())
		return
	}

	fi, err := file.Readdir(-1)
	if err != nil {
		log.Println(err.Error())
		return
	}

	for _, f := range fi {
		fr := MakeFileRef(webPath, f)
		fc.FileRefs = append(fc.FileRefs, fr)
		if f.IsDir() {
			fc.doRefreshDirectory(path.Join(realPath, f.Name()), path.Join(webPath, f.Name()))
		}
	}
}

func (fc *FileCache) doRefresh() {
	fc.FileRefs = []FileRef{}
	fc.doRefreshDirectory(fc.Path, "/")
}

func (fc *FileCache) doSearch(regex string, responseChannel chan []FileRef) {
	re, err := regexp.Compile(regex)
	if err != nil {
		// if it failed just close the response channel to indicate that
		// it failed
		close(responseChannel)
	}

	matchedFiles := []FileRef{}

	for _, f := range fc.FileRefs {
		if re.MatchString(f.Path) {
			matchedFiles = append(matchedFiles, f)
		}
	}

	responseChannel <- matchedFiles
	close(responseChannel)
}

func (fc *FileCache) refresh() {
	task := fcTask{}
	task.action = "refresh"
	fc.RecvChannel <- task
}

func NewFileCache(path string) *FileCache {
	fc := FileCache{}
	fc.Path = path
	fc.RecvChannel = make(chan fcTask)
	go fc.daemon()
	fc.refresh()

	// start the refresh daemon. just pings the filecache
	// daemon every minute to refresh its cache
	go func() {
		for {
			// if the channel is closed return
			if fc.Closed {
				return
			}
			time.Sleep(time.Minute)
			fc.refresh()
		}
	}()

	return &fc
}

func (fc *FileCache) Search(regex string) ([]FileRef, error) {
	task := fcTask{}
	task.action = "search"
	filerefChannel := make(chan []FileRef)
	task.args = []interface{}{regex, filerefChannel}

	fc.RecvChannel <- task

	timeout := make(chan bool, 1)
	go func() {
		// timeout after 5 seconds
		time.Sleep(5 * time.Second)
	}()
	select {
	case resRefs, ok := <-filerefChannel:
		if !ok {
			return nil, fmt.Errorf("Search \"%s\" failed", regex)
		}

		return resRefs, nil
	case <-timeout:
		return nil, fmt.Errorf("Search \"%s\" timed out", regex)
	}

	return nil, fmt.Errorf("Search \"%s\" failed. FileCache daemon not running")
}

func (fc *FileCache) Close() {
	fc.Closed = true
	close(fc.RecvChannel)
}
