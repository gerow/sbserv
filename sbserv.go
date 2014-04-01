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
	"syscall"
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, f := range fi {
		fmt.Fprintf(w, "%s\n", f.Name())
	}
}

func handleDownloadDir(file *os.File, p string, w http.ResponseWriter, r *http.Request) {
	fi, err := file.Readdir(-1)

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//fmt.Fprintf(w, "we're going to try to give you a zip now!")

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, fiEntry := range fi {
		f, err := os.Open(path.Join(p, fiEntry.Name()))
		if err != nil {
			// if we have any trouble with a file just skip it and log the error
			log.Println(err.Error())
			continue
		}

		defer f.Close()

		fileWriter, err := zw.Create(fiEntry.Name())
		if err != nil {
			log.Println(err.Error())
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
		log.Println("Received request for file outside serve root.")
		http.Error(w, "Refusing to serve path outside serve root.", http.StatusBadRequest)
		return
	}

	file, err := os.Open(p)
	defer file.Close()

	if err != nil {
		log.Println(err.Error())
		if (err.(*os.PathError)).Err == syscall.ENOENT {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	fi, err := file.Stat()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		log.Println("Received attempt to serve non-regular file")
		http.Error(w, "Refusing to read a non-regular file.", http.StatusBadRequest)
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
