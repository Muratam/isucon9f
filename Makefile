isutrain: $(wildcard *.go)
	go build -o $@
	sudo systemctl restart isutrain-go
