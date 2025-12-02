
# Copyright © 2023 Thomas von Dein

# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with this program. If not, see <http://www.gnu.org/licenses/>.


#
# no need to modify anything below
version   = $(shell egrep "= .v" cfg/config.go | cut -d'=' -f2 | cut -d'"' -f 2)
archs     = android darwin freebsd linux netbsd openbsd windows
PREFIX    = /usr/local
UID       = root
GID       = 0
BRANCH    = $(shell git branch --show-current)
COMMIT    = $(shell git rev-parse --short=8 HEAD)
BUILD     = $(shell date +%Y.%m.%d.%H%M%S) 
VERSION  := $(if $(filter $(BRANCH), development),$(version)-$(BRANCH)-$(COMMIT)-$(BUILD),$(version))
BR_MAIN  := $(if $(filter $(BRANCH), main),"main","")
HAVE_POD := $(shell pod2text -h 2>/dev/null)
HAVE_LINT:= $(shell golangci-lint -h 2>/dev/null)
DAEMON   := ephemerupd
CLIENT   := upctl
DATE      = $(shell date +%Y-%m-%d)


all: cmd/formtemplate.go lint buildlocal buildlocalctl

lint:
ifdef HAVE_LINT
	golangci-lint run
endif

buildlocalctl:
	make -C upctl

buildlocal:
#	go build -ldflags "-X 'codeberg.org/scip/ephemerup/cfg.VERSION=$(VERSION)'" -o $(DAEMON)
	CGO_LDFLAGS='-static' go build -tags osusergo,netgo -ldflags "-extldflags=-static -s -X 'codeberg.org/scip/ephemerup/cfg.VERSION=$(VERSION)'" -o $(DAEMON)

buildimage: clean
	docker-compose --verbose build

release:
ifdef BR_MAIN
	git tag -a $(version) -m "$(version) released on $(DATE)"
	git push origin --tags
	./mkrel.sh $(DAEMON) $(CLIENT) $(version)
	gh release create $(version) --generate-notes releases/*
else
	@echo "Cannot create release on branch $(BRANCH), checkout main and retry!"
endif

install: buildlocal
	install -d -o $(UID) -g $(GID) $(PREFIX)/bin
	install -o $(UID) -g $(GID) -m 555 $(DAEMON) $(PREFIX)/sbin/

cleanctl:
	make -C upctl clean

clean: cleanctl
	rm -rf releases coverage.out $(DAEMON)

test:
	go test -v ./...
#	bash t/test.sh

singletest:
	@echo "Call like this: ''make singletest TEST=TestX1 MOD=lib"
	go test -run $(TEST) codeberg.org/scip/ephemerup/$(MOD)

cover-report:
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out

show-versions: buildlocal
	@echo "### ephemerupd version:"
	@./ephemerupd --version

	@echo
	@echo "### go module versions:"
	@go list -m all

	@echo
	@echo "### go version used for building:"
	@grep -m 1 go go.mod

goupdate:
	go get -t -u=patch ./...

cmd/%.go: templates/%.html
	echo "package cmd" > cmd/$*.go
	echo >> cmd/$*.go
	echo "const formtemplate = \`" >> cmd/$*.go
	cat templates/$*.html >> cmd/$*.go
	echo "\`" >> cmd/$*.go
