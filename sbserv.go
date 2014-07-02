package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	id3 "github.com/gerow/id3-go"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

type FileRef struct {
	Path      string
	Name      string
	ModTime   string
	Size      int64
	Glyphicon string
	Type      string
	IsDir     bool   `json:"-"`
	VideoType string `json:"-"`
	Extra     map[string]interface{}
}

type Page struct {
	Path     string
	FileRefs []FileRef `json:"Files"`
	VHash    string
}

var cwd string
var dirListingTemplate *template.Template
var vhash string
var fileServerHandler http.Handler
var fileCache FileCache

type ByName []FileRef

func (a ByName) Len() int {
	return len(a)
}

func (a ByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

func MakeFileRef(leadingPath string, f os.FileInfo) FileRef {
	var fr FileRef
	fr.Name = f.Name()
	fr.Path = path.Join(leadingPath, f.Name())
	const layout = "2006-01-02 15:04:05"
	fr.ModTime = string(f.ModTime().Format(layout))
	fr.Size = f.Size()
	fr.Glyphicon = "glyphicon-file"
	fr.Extra = make(map[string]interface{})

	if f.Mode().IsDir() {
		fr.Glyphicon = "glyphicon-folder-open"
		fr.IsDir = true
		fr.Type = "directory"
	} else {
		fr.IsDir = false
		fr.Type = "file"
		ext := filepath.Ext(fr.Path)
		switch {
		case ext == ".mp3":
			fallthrough
		case ext == ".ogg":
			fallthrough
		case ext == ".flac":
			fr.Glyphicon = "glyphicon-music"
		case ext == ".jpg":
			fallthrough
		case ext == ".jepg":
			fallthrough
		case ext == ".png":
			fallthrough
		case ext == ".bmp":
			fallthrough
		case ext == ".gif":
			fr.Glyphicon = "glyphicon-picture"
		case ext == ".avi":
			fallthrough
		case ext == ".flv":
			fallthrough
		case ext == ".mpeg":
			fallthrough
		case ext == ".mpg":
			fallthrough
		case ext == ".mpe":
			fallthrough
		case ext == ".ogv":
			fr.Glyphicon = "glyphicon-film"
		case ext == ".mkv":
			fr.VideoType = "video/webm"
			fr.Glyphicon = "glyphicon-film"
		case ext == ".mov":
			fallthrough
		case ext == ".m4v":
			fallthrough
		case ext == ".mp4":
			fr.VideoType = "video/mp4"
			fr.Glyphicon = "glyphicon-film"
		case ext == ".zip":
			fallthrough
		case ext == ".tar":
			fallthrough
		case ext == ".gz":
			fallthrough
		case ext == ".rar":
			fr.Glyphicon = "glyphicon-compressed"
		case ext == ".epub":
			fallthrough
		case ext == ".mobi":
			fallthrough
		case ext == ".pdf":
			fr.Glyphicon = "glyphicon-book"
		}
		if ext == ".mp3" {
			mp3File, err := id3.OpenReadOnly(path.Join(cwd, fr.Path))
			if err == nil {
				defer mp3File.Close()
				fr.Extra["id3"] = struct {
					Title    string
					Artist   string
					Album    string
					Year     string
					Genre    string
					Comments []string
				}{
					mp3File.Title(),
					mp3File.Artist(),
					mp3File.Album(),
					mp3File.Year(),
					mp3File.Genre(),
					mp3File.Comments(),
				}
			}
		}
	}

	return fr
}

func handleDir(file *os.File, p string, w http.ResponseWriter, r *http.Request) {
	// Read the directory
	fi, err := file.Readdir(-1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var page Page
	page.Path = r.URL.Path
	page.VHash = vhash
	for _, f := range fi {
		fr := MakeFileRef(r.URL.Path, f)
		page.FileRefs = append(page.FileRefs, fr)
	}

	sort.Sort(ByName(page.FileRefs))

	if r.FormValue("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		jsonForm, err := json.Marshal(page)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, string(jsonForm))
	} else {
		dirListingTemplate.Execute(w, page)
	}
}

func writeDir(file *os.File, p string, prefix string, zw *zip.Writer, w http.ResponseWriter) {
	log.Printf("Creating dir from %s with prefix %s\n", p, prefix)

	fi, err := file.Readdir(-1)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, fiEntry := range fi {
		f, err := os.Open(path.Join(p, fiEntry.Name()))
		if err != nil {
			log.Println(err.Error())
			continue
		}
		defer f.Close()

		if fiEntry.IsDir() {
			log.Printf("Creating subdirectory for %v\n", fiEntry)
			writeDir(f, path.Join(p, fiEntry.Name()), path.Join(prefix, fiEntry.Name()), zw, w)
			continue
		}

		fileWriter, err := zw.Create(path.Join(prefix, fiEntry.Name()))
		if err != nil {
			log.Println(err.Error())
			continue
		}

		io.Copy(fileWriter, f)
	}
}

func handleDownloadDir(file *os.File, p string, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")

	zw := zip.NewWriter(w)
	defer zw.Close()

	f, err := os.Open(p)
	defer f.Close()

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeDir(f, p, "", zw, w)
}

func handleFile(file *os.File, p string, w http.ResponseWriter, r *http.Request, fi os.FileInfo) {
	//io.Copy(w, file)
	//fileServerHandler.ServeHTTP(w, r)
	etagBytes := sha256.Sum256([]byte(fi.ModTime().String()))
	etag := hex.EncodeToString(etagBytes[:len(etagBytes)])
	w.Header().Set("ETag", etag)
	http.ServeContent(w, r, p, fi.ModTime(), file)
}

func handleStatic(p string, w http.ResponseWriter, r *http.Request) {
	p = strings.TrimPrefix(p, "/_static/"+vhash)

	log.Printf("Got request for static asset %s", p)

	assetPath := path.Join("data/static/", p)
	log.Printf("Using path %s", assetPath)

	assetBytes, err := Asset(assetPath)
	if err != nil {
		log.Println("Received request for static file we don't have")
		http.Error(w, "No such static asset", http.StatusNotFound)
		return
	}

	log.Printf("Using extension %s", filepath.Ext(p))

	switch ext := filepath.Ext(p); {
	case ext == ".css":
		w.Header().Set("Content-Type", "text/css")
	case ext == ".js":
		w.Header().Set("Content-Type", "text/javascript")
	case ext == ".png":
		w.Header().Set("Content-Type", "image/png")
	}

	// Don't ever expire
	w.Header().Set("Cache-Control", "public")
	// Or at least don't expire until the AI machines take over. They
	// can deal with fixing this.
	w.Header().Set("Expires", "Sun, 17-Jan-2038 19:14:07 GMT")

	fmt.Fprint(w, string(assetBytes))
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	queries, ok := r.Form["query"]
	if !ok {
		log.Println("Received search request without query argument")
		http.Error(w, "Search requires query argument", http.StatusBadRequest)
		return
	}

	if len(queries) != 1 {
		log.Println("Received multiple query arguments")
		http.Error(w, "Search requires one and only one query argument", http.StatusBadRequest)
		return
	}

	query := queries[0]

	log.Printf("got search request for \"%s\"", query)
	fileRefs, err := fileCache.Search(query)
	if err != nil {
		log.Println(err)
		http.Error(w, "invalid regex", http.StatusBadRequest)
		return
	}

	page := Page{
		Path:     "/_search query: " + query,
		FileRefs: fileRefs,
		VHash:    vhash,
	}

	if r.FormValue("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		jsonForm, err := json.Marshal(page)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, string(jsonForm))
		return
	}

	dirListingTemplate.Execute(w, page)
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("Dumping header values:")
	for k, v := range r.Header {
		log.Printf("%s - %s\n", k, v)
	}

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

	// determine if this is a search request
	if r.URL.Path == "/_search" {
		handleSearch(w, r)
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
		handleFile(file, p, w, r, fi)
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

	vhashBytes, err := Asset("data/version_hash")
	if err != nil {
		log.Fatal(err)
	}

	vhash = string(vhashBytes)

	if len(os.Args) != 2 {
		log.Fatal("must specify bind address")
	}

	bindAddress := os.Args[1]

	fileServerHandler = http.FileServer(http.Dir(cwd))

	// Start the file cache daemon
	fileCache = *NewFileCache(cwd)

	matchedRef, err := fileCache.Search("\\.mp3")
	log.Printf("\\.mp3 matched")
	for _, f := range matchedRef {
		log.Printf(f.Path)
		//log.Printf("Artist: ", f.Extra["id3"].Artist)
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(bindAddress, nil)
}
