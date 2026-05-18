default: build

build:
	go build -trimpath ./...

install:
	go install -trimpath ./...

fmt:
	@git ls-files '*.go' ':!:generated/**' | xargs gofmt -w

test:
	go test ./...

vet:
	@packages="$$(go list ./... | grep -v '^github.com/nebius/terraform-provider-nebius/generated\($$\|/\)')"; \
	go vet $$packages

generate:
	cd tools && go generate ./...

generate-check: generate
	git diff --exit-code

docs-validate:
	cd tools && go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest validate --provider-dir .. --provider-name nebius

release-snapshot:
	goreleaser release --clean --snapshot

.PHONY: build install fmt test vet generate generate-check docs-validate release-snapshot
