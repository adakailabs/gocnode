export VERSION
GOLANG-CI := $(shell  command -v  golangci-lint version  2> /dev/null)
GOLANG-CI-CONFIG_URI=https://raw.githubusercontent.com/lagarciag/dotfiles/master/go/.golangci.yaml
PACKAGE := ""

tests: golang-ci .golangci.yaml
	#golangci-lint run --fix --tests  -v 
	#go test github.hpe.com/hpe-networking/${PACKAGE}/...

GTAG=$(shell git describe --always)
GVERSION=${VERSION}-${GTAG}

.PHONY: golang-ci
golang-ci:
ifndef GOLANG-CI
	@echo asdf
	@echo ${DOT}
	@echo "need to install golang-ci to tests command to run" 
	sudo curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/bin v1.36.0
endif	

.golangci.yaml:
	wget ${GOLANG-CI-CONFIG_URI}

.PHONY: gocnode
gocnode:
	env GOOS="linux" GOARCH="amd64" go build -o gocnode_amd64 -ldflags "-s -w -X main.Version='${GVERSION}-amd64'" -v gocnode.go 
	env GOOS="linux" GOARCH="arm64" go build -o gocnode_arm64 -ldflags "-s -w -X main.Version='${GVERSION}-arm64'" -v gocnode.go
	#go build -o gocnode -ldflags "-s -w -X main.Version='${GVERSION}-amd64'" -v gocnode.go
tversion:
	@echo ${GVERSION}

clean:
	rm gocnode_*
