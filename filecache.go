package main

import (
	"log"
	"os"
	"path"
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
			fc.doSearch()
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

func (fc *FileCache) doSearch() {

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

func (fc *FileCache) Search() {
}

func (fc *FileCache) Close() {
	fc.Closed = true
	close(fc.RecvChannel)
}
