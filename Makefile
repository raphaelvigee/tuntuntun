run-sample-box:
	cd cmd/example && GOOS=linux GOARCH=arm64 go build -o ../../sample-box/tuntuntun .
	cd sample-box && docker build -t tuntuntun-samplebox .
	docker run --rm -p 5201:5201 tuntuntun-samplebox

run-sample-box-server:
	go run ./cmd/example/main.go server -addr=:8888 -transport=ws -remote-addrs=:22,:8080,:5201
