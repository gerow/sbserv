package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type FileRef struct {
	Path string
	Name string
}

type Page struct {
	Path     string
	FileRefs []FileRef
}

var cwd string

func handleDir(file *os.File, p string, w http.ResponseWriter, r *http.Request) {
	// Read the directory
	fi, err := file.Readdir(-1)

	if err != nil {
		fmt.Fprintf(w, "failed")
		return
	}

	for _, f := range fi {
		fmt.Fprintf(w, "%s\n", f.Name())
	}
}

func handleDownloadDir(file *os.File, p string, w http.ResponseWriter, r *http.Request) {
	fi, err := file.Readdir(-1)

	if err != nil {
		fmt.Fprintf(w, "failed to handle dl dir")
		return
	}

	//fmt.Fprintf(w, "we're going to try to give you a zip now!")

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, fiEntry := range fi {
		f, err := os.Open(path.Join(p, fiEntry.Name()))
		if err != nil {
			continue
		}

		defer f.Close()

		fileWriter, err := zw.Create(fiEntry.Name())
		if err != nil {
			continue
		}

		io.Copy(fileWriter, f)
	}
}

func handleFile(file *os.File, p string, w http.ResponseWriter, r *http.Request) {
	io.Copy(w, file)
}

func handler(w http.ResponseWriter, r *http.Request) {
	p := path.Join(cwd, r.URL.Path)
	p = path.Clean(p)
	if !strings.HasPrefix(p, cwd) {
		return
	}

	file, err := os.Open(p)
	defer file.Close()

	if err != nil {
		fmt.Fprintf(w, "Failed to open file.")
		return
	}

	fi, err := file.Stat()
	if err != nil {
		fmt.Fprintf(w, "Failed to stat file.")
		return
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		if r.FormValue("dldir") == "true" {
			handleDownloadDir(file, p, w, r)
		} else {
			handleDir(file, p, w, r)
		}
	case mode.IsRegular():
		handleFile(file, p, w, r)
	default:
		return
	}
}

func main() {
	var err error

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
		return
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
