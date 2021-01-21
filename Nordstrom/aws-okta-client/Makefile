# Goals:
# - user can build binaries on their system without having to inst-platforms special tools
# - user can fork the canonical repo and expect to be able to run Gitlab CI checks
#
# This makefile is meant for humans
# example: make release-stage -e VERSION=0.0.4.1-dev RELEASE="nordstrom aws-okta cli - PC-6864-development"

#$(shell git describe --tags --always --dirty="-dev")
# use make -e VERSION="<version info>" RELEASE="<release info>" to override presets 
# ***NOTE: tack -DEV to version to ensure "dev build verbage" triggers in update checker  
#VERSION := 0.0.1
#RELEASE := "NORDSTROM AWS-OKTA CLI"
JSON    := jq -n --arg V "$(VERSION)" --arg R "$(RELEASE)" '{version: $V, message: $R}'
LDFLAGS := -ldflags='-X "main.Version=$(VERSION)"'

release-dev:  clean all-platforms stage deploy-dev

release-prod:  clean all-platforms stable deploy-prod

all-platforms: clean release/aws-okta-$(VERSION)-darwin-amd64 release/aws-okta-$(VERSION)-linux-amd64 release/aws-okta-$(VERSION)-windows-amd64 

dev:  clean all-platforms development #edit for use case as needed
prod: clean all-platforms stable #edit for use case as needed

test:
	go test -v ./tests

clean:
	rm -rf ./release

release/:
	mkdir -p release

release/aws-okta-$(VERSION)-darwin-amd64: |
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $@

release/aws-okta-$(VERSION)-linux-amd64: |
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $@

release/aws-okta-$(VERSION)-windows-amd64: |
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $@.exe
	

stable:
	echo v$(VERSION) > release/stable.txt
	echo '{"version": "$(VERSION)", "message": "$(RELEASE)"}' > release/stable.json

development:
	echo v$(VERSION) > release/development.txt
	echo '{"version": "$(VERSION)", "message": "$(RELEASE)"}' > release/development.json

deploy-prod:
	gsutil cp release/aws-okta-$(VERSION)-darwin-amd64 "gs://aws-okta/release/v$(VERSION)/bin/darwin/x64/aws-okta"
	gsutil cp release/aws-okta-$(VERSION)-linux-amd64 "gs://aws-okta/release/v$(VERSION)/bin/linux/x64/aws-okta"
	gsutil cp release/aws-okta-$(VERSION)-windows-amd64.exe "gs://aws-okta/release/v$(VERSION)/bin/win/x64/aws-okta.exe"
	gsutil rm -r "gs://aws-okta/release/LATEST"
	gsutil cp -r "gs://aws-okta/release/v$(VERSION)" "gs://aws-okta/release/LATEST"
	sleep 10
	gsutil cp release/stable.txt "gs://aws-okta/release/stable.txt"
	gsutil cp release/stable.json "gs://aws-okta/release/stable.json"


deploy-dev:
	gsutil cp release/aws-okta-$(VERSION)-darwin-amd64 "gs://aws-okta/development/v$(VERSION)/bin/darwin/x64/aws-okta-dev"
	gsutil cp release/aws-okta-$(VERSION)-linux-amd64 "gs://aws-okta/development/v$(VERSION)/bin/linux/x64/aws-okta-dev"
	gsutil cp release/aws-okta-$(VERSION)-windows-amd64.exe "gs://aws-okta/development/v$(VERSION)/bin/win/x64/aws-okta-dev.exe"
	gsutil rm -r "gs://aws-okta/development/LATEST"
	gsutil cp -r "gs://aws-okta/development/v$(VERSION)" "gs://aws-okta/development/LATEST"
	sleep 10
	gsutil cp release/development.txt "gs://aws-okta/development/development.txt"
	gsutil cp release/development.json "gs://aws-okta/development/development.json"

.PHONY: test clean all-platforms