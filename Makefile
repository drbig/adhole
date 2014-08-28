all: adhole genlist

adhole/adhole: adhole/main.go adhole/sigwait_unix.go adhole/sigwait_windows.go
	cd adhole; \
	gofmt -w *.go; \
	go build .

genlist/genlist: genlist/main.go genlist/sources.go
	cd genlist; \
	gofmt -w *.go; \
	go build .

adhole: adhole/adhole
genlist: genlist/genlist
.PHONY: adhole
.PHONY: genlist
