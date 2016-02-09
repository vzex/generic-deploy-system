.PHONY: local server remote clean

server:
	go build -o run_$@ main.go server.go
remote:
	go build -o run_$@ main.go remote.go
clean:
	@rm -f run_*
