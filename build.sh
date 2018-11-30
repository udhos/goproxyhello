#!/bin/sh

gofmt -s -w ./goproxyhello
go tool fix ./goproxyhello
go tool vet ./goproxyhello

#hash gosimple && gosimple ./goproxyhello
#hash golint && golint ./goproxyhello
#hash staticcheck && staticcheck ./goproxyhello

go test ./goproxyhello
go install ./goproxyhello
