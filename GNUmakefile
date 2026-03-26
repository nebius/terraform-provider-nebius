default: build

build:
	go build -trimpath ./...

install:
	go install -trimpath ./...

fmt:
	gofmt -w .

test:
	go test ./...

generate:
	cd tools && go generate ./...

docs-validate:
	cd tools && go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest validate --provider-dir .. --provider-name nebius

release-snapshot:
	goreleaser release --clean --snapshot

.PHONY: build install fmt test generate docs-validate release-snapshot
