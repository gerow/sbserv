DATA_DIR   = data
DATA_FILES = $(DATA_DIR)/dir_listing.html

all: bindata.go
	go build

bindata.go: $(DATA_FILES)
	go get github.com/jteeuwen/go-bindata/...
	go-bindata data/

clean:
	-rm -f bindata.go
	-rm -f sbserv
