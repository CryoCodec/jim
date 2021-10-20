PLATFORMS := linux/amd64 darwin/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

.PHONY: release $(PLATFORMS)
release: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -o 'build/$(os)-$(arch)/jimClient' bin/jimClient/main.go
	GOOS=$(os) GOARCH=$(arch) go build -o 'build/$(os)-$(arch)/jimServer' bin/jimServer/main.go
	cp static/* 'build/$(os)-$(arch)'
	mkdir -p dist
	cd build && tar -zcvf ../dist/jim-$(os)-$(arch).tar.gz $(os)-$(arch)

.PHONY: build
build: build-client build-server
	cp static/* build/local/

.PHONY: build-client
build-client:
	go build -o build/local/jimClient bin/jimClient/main.go

.PHONY: build-server
build-server:
	go build -o build/local/jimServer bin/jimServer/main.go

.PHONY: clean
clean:
	rm -rf build/
	rm -rf dist/

install-requirements-mac-x86_64: _download-protoc-mac-x86_64 _unzip-protoc _locate-protoc _cleanup-tmp _goget-grpc
install-requirements-linux-x86_64: _download-protoc _unzip-protoc _locate-protoc _cleanup-tmp _goget-grpc

protoc: _gen-go-out

_download-protoc-mac-x86_64:
	mkdir -p tmp && \
	cd tmp && \
	curl -L https://github.com/protocolbuffers/protobuf/releases/download/v3.18.1/protoc-3.18.1-osx-x86_64.zip --output protoc.zip

_download-protoc-linux-x86_64:
	mkdir -p tmp && \
	cd tmp && \
	curl -L https://github.com/protocolbuffers/protobuf/releases/download/v3.18.1/protoc-3.18.1-linux-x86_64.zip --output protoc.zip


_unzip-protoc:
	cd tmp && \
	unzip ./protoc.zip -d protoc

_locate-protoc:
	mkdir -p bin && \
	rm -rf bin/protoc && \
	cd tmp && \
	mv -f ./protoc/ ../tools

_cleanup-tmp:
	rm -rf ./tmp

_goget-grpc:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go

_gen-go-out:
	mkdir -p internal
	tools/bin/protoc --go_out=plugins=grpc:./ proto/*.proto
