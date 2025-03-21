default:
	go build -o fetch cmd/fetch/main.go
	go build -o chart cmd/chart/main.go

clean:
	rm -f fetch
	rm -f chart