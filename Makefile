DATA_DIR    = data

DATA_FILES  = $(DATA_DIR)/templates/dir_listing.html
DATA_FILES += $(DATA_DIR)/static/css/bootstrap-responsive.css
DATA_FILES += $(DATA_DIR)/static/css/bootstrap-responsive.min.css
DATA_FILES += $(DATA_DIR)/static/css/bootstrap.css
DATA_FILES += $(DATA_DIR)/static/css/bootstrap.min.css
DATA_FILES += $(DATA_DIR)/static/css/bootstrap.min.css
DATA_FILES += $(DATA_DIR)/static/img/glyphicons-halflings-white.png
DATA_FILES += $(DATA_DIR)/static/img/glyphicons-halflings.png
DATA_FILES += $(DATA_DIR)/static/js/bootstrap.js
DATA_FILES += $(DATA_DIR)/static/js/bootstrap.min.js

all: bindata.go
	go build

bindata.go: $(DATA_FILES)
	go get github.com/jteeuwen/go-bindata/...
	go-bindata data/...

clean:
	-rm -f bindata.go
	-rm -f sbserv
