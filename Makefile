path=$(shell pwd)
all: dep build
dep:
	go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct
	go mod tidy -compat=1.17
build:
	CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o app .