.PHONY: local server remote clean

server:
	go build -ldflags "-s -w" -o run_$@ main.go server.go
remote:
	go build -ldflags "-s -w" -o run_$@ main.go remote.go
clean:
	@rm -f run_*
