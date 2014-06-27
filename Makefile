DATA_DIR    = data

DATA_FILES  = $(DATA_DIR)/templates/dir_listing.html
DATA_FILES += $(DATA_DIR)/version_hash
DATA_FILES += $(DATA_DIR)/static/js/jquery.tablesorter.min.js
DATA_FILES += $(DATA_DIR)/static/js/dir_listing.js
DATA_FILES += $(DATA_DIR)/static/css/dir_listing.css

GO_FILES    = bindata.go
GO_FILES   += sbserv.go
GO_FILES	 += filecache.go

all: sbserv sbserv.debug


sbserv: $(GO_FILES)
	go get
	go build

sbserv.debug: $(GO_FILES)
	go get
	go build -o sbserv.debug -gcflags "-N -l"

$(DATA_DIR)/version_hash: .git
	git rev-parse HEAD > $(DATA_DIR)/version_hash

bindata.go: $(DATA_FILES)
	go get github.com/jteeuwen/go-bindata/...
	$(GOPATH)/bin/go-bindata data/...

clean:
	-rm -f bindata.go
	-rm -f sbserv sbserv.debug
	-rm -f sbserv-linux-amd64 sbserv-linux-386 sbserv-freebsd-amd64 sbserv-freebsd-386 sbserv-darwin-amd64 sbserv-darwin-386

cross-compile: sbserv-linux-amd64 sbserv-linux-386 sbserv-freebsd-amd64 sbserv-freebsd-386 sbserv-darwin-amd64 sbserv-darwin-386

sbserv-linux-amd64: $(GO_FILES)
	GOOS=linux GOARCH=amd64 go build -o sbserv-linux-amd64

sbserv-linux-386: $(GO_FILES)
	GOOS=linux GOARCH=386 go build -o sbserv-linux-386

sbserv-freebsd-amd64: $(GO_FILES)
	GOOS=freebsd GOARCH=amd64 go build -o sbserv-freebsd-amd64

sbserv-freebsd-386: $(GO_FILES)
	GOOS=freebsd GOARCH=386 go build -o sbserv-freebsd-386

sbserv-darwin-amd64: $(GO_FILES)
	GOOS=darwin GOARCH=amd64 go build -o sbserv-darwin-amd64

sbserv-darwin-386: $(GO_FILES)
	GOOS=darwin GOARCH=386 go build -o sbserv-darwin-386

