default:
	go build -o fetch cmd/fetch/main.go
	go build -o chart cmd/chart/main.go
	go build -o png ./cmd/png

clean:
	rm -f fetch
	rm -f chart
	rm -f png
