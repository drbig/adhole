NAME=adhole
PARTS=main.go sigwait_unix.go sigwait_windows.go

$(NAME): $(PARTS)
	gofmt -w $(PARTS)
	go build .

docs: doc.go
	godoc -notes="BUG|TODO" .

test: $(PARTS)
	go tool vet -all -v .

.PHONY: test
.PHONY: docs
