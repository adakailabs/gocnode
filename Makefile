GOLANG-CI := $(shell  command -v  golangci-lint version  2> /dev/null)
GOLANG-CI-CONFIG_URI=https://raw.githubusercontent.com/lagarciag/dotfiles/master/go/.golangci.yaml
PACKAGE := ""

tests: golang-ci .golangci.yaml
	golangci-lint run --fix --tests  -v 
	#go test github.hpe.com/hpe-networking/${PACKAGE}/...

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
