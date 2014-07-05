NAME=adhole
PARTS=main.go

$(NAME): $(PARTS)
	gofmt -w $(PARTS)
	go build .

test: $(PARTS)
	go tool vet -all -v .

docs: doc.go
	godoc -notes="BUG|TODO" .

.PHONY: test
