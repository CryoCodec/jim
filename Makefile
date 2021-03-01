build: 
	go build -o build/jimServer bin/jimServer/main.go
	go build -o build/jim bin/jimClient/main.go

build-client:
	go build -o build/jim bin/jimClient/main.go

build-server:
	go build -o build/jimServer bin/jimServer/main.go

run-client:
	go run bin/jimClient/main.go

run-server:
	go run bin/jimClient/main.go

clean:
	rm -rf build/