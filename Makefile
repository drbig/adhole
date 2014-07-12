NAME=adhole
PARTS=main.go

$(NAME): $(PARTS)
	gofmt -w $(PARTS)
	go build .

docs: doc.go
	godoc -notes="BUG|TODO" .

test: $(PARTS)
	go tool vet -all -v .

.PHONY: test
.PHONY: docs
