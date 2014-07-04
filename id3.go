package main

import (
	id3 "github.com/gerow/id3-go"
)

type Id3Cache struct {
	cache map[string]*Id3Extra
}

func NewId3Cache() *Id3Cache {
	return &Id3Cache{
		cache: make(map[string]*Id3Extra),
	}
}

func (c *Id3Cache) Get(path string) (*Id3Extra, error) {
	extra, ok := c.cache[path]
	if ok {
		return extra, nil
	}

	mp3File, err := id3.OpenReadOnly(path)
	if err != nil {
		return nil, err
	}

	defer mp3File.Close()

	extra = &Id3Extra{
		mp3File.Title(),
		mp3File.Artist(),
		mp3File.Album(),
		mp3File.Year(),
		mp3File.Genre(),
		mp3File.Comments(),
	}

	c.cache[path] = extra

	return extra, nil
}
