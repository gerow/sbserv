package main

import (
	"archive/zip"
	"fmt"
	"html/template"
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
var dirListingTemplate *template.Template

func handleDir(file *os.File, p string, w http.ResponseWriter, r *http.Request) {
	// Read the directory
	fi, err := file.Readdir(-1)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var page Page
	page.Path = r.URL.Path
	for _, f := range fi {
		//fmt.Fprintf(w, "%s\n", f.Name())
		var fr FileRef
		fr.Name = f.Name()
		fr.Path = path.Join(r.URL.Path, f.Name())
		page.FileRefs = append(page.FileRefs, fr)
	}

	dirListingTemplate.Execute(w, page)
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

func handleStatic(p string, w http.ResponseWriter, r *http.Request) {
	p = strings.TrimPrefix(p, "/_static/")

	log.Printf("Got request for static asset %s", p)

	assetPath := path.Join("data/static/", p)
	log.Printf("Using path %s", assetPath)

	assetBytes, err := Asset(assetPath)
	if err != nil {
		log.Println("Received request for static file we don't have")
		http.Error(w, "No such static asset", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, string(assetBytes))
}

func handler(w http.ResponseWriter, r *http.Request) {
	p := path.Join(cwd, r.URL.Path)
	p = path.Clean(p)
	if !strings.HasPrefix(p, cwd) {
		log.Println("Received request for file outside serve root.")
		http.Error(w, "Refusing to serve path outside serve root.", http.StatusBadRequest)
		return
	}

	// determine if this is a request for assets
	if strings.HasPrefix(r.URL.Path, "/_static/") {
		handleStatic(r.URL.Path, w, r)
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
		log.Printf("Handling %s as a directory\n", p)
		if r.FormValue("dldir") == "true" {
			handleDownloadDir(file, p, w, r)
		} else {
			handleDir(file, p, w, r)
		}
	case mode.IsRegular():
		log.Printf("Handling %s as a regular file\n", p)
		handleFile(file, p, w, r)
	default:
		log.Println("Received attempt to serve non-regular file")
		http.Error(w, "Refusing to read a non-regular file.", http.StatusBadRequest)
		return
	}
}

func main() {
	var err error

	log.Printf("starting")

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Parse the dir listing template
	dirListingBytes, err := Asset("data/templates/dir_listing.html")
	if err != nil {
		log.Fatal(err)
	}

	dirListingTemplate, err = template.New("dir_listing.html").Parse(string(dirListingBytes))
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
