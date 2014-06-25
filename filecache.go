package main

import (
	"time"
)

type FileCache struct {
	Path        string
	RecvChannel chan fcTask
	Closed      bool
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

func (fc *FileCache) doRefresh() {

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
