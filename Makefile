# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Print this help message.
	@echo; echo = Targets =
	@grep -E '^\w+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z_]+: ' Makefile | sed 's/^## \([A-Z_]*\): \(.*\)/\1#\2/' | column -s'#' -t

# The following variables can be overridden by environment variables or on the `make` command line

## VERSION: semantic version for releases, based on "git describe" for work in development (not semver).
VERSION?=$(or $(shell git describe --dirty 2>/dev/null | cut -d- -f1,2,4- | sed 's/-/_dev_/'),$(file <$(VERSION_TXT)))
## IMG: Name of image to build or deploy, without version tag.
IMG?=quay.io/korrel8r/korrel8r
## TAG: Image tag, defaults to $(VERSION)
TAG?=$(VERSION)
## OVERLAY: Name of kustomize directory in config/overlays to use for `make deploy`.
OVERLAY?=dev
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(shell which podman || which docker)

check: generate lint test ## Lint and test code.

all: check install _site image ## Build everything.

clean: # Warning: runs `git clean -dfx` and removes checked-in generated files.
	rm -vrf _site docs/zz_*.adoc pkg/api/zz_docs /cmd/korrel8r/version.txt
	git clean -dfx

tools: ## Install tools for generating, linting nad testing locally.
	go install github.com/go-swagger/go-swagger/cmd/swagger@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install sigs.k8s.io/kind@latest

VERSION_TXT=cmd/korrel8r/version.txt

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match VERSION
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

generate: $(VERSION_TXT) pkg/api/zz_docs $(shell find -name '*.go') ## Generate code.
	hack/copyright.sh
	go mod tidy

pkg/api/zz_docs: $(wildcard pkg/api/*.go pkg/korrel8r/*.go)
	@mkdir -p $(dir $@)
	swag init -q -g pkg/api/api.go -o $@
	swag fmt pkg/api
	@touch $@

lint: $(VERSION_TXT) ## Run the linter to find and fix code style problems.
	golangci-lint run --fix

install: $(VERSION_TXT) ## Build and install the korrel8r binary locally in $GOBIN.
	go install -tags netgo ./cmd/korrel8r

test: ## Run all tests, requires a cluster.
	$(MAKE) TEST_NO_SKIP=1 test-skip
test-skip: $(VERSION_TXT) ## Run all tests but skip those requiring a cluster if not logged in.
	go test -timeout=1m -race ./...

cover: ## Run tests and show code coverage in browser.
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

CONFIG=etc/korrel8r/korrel8r.yaml
GOBIN=$(or $(go env GOBIN),$(HOME)/go/bin)
run: install ## Install and run `korrel8r web` using configuration in ./etc/korrel8r
	$(GOBIN)/korrel8r web -c $(CONFIG)

IMAGE=$(IMG):$(TAG)
image: $(VERSION_TXT) ## Build and push image. IMG must be set to a writable image repository.
	$(IMGTOOL) build --tag=$(IMAGE) .
	$(IMGTOOL) push -q $(IMAGE)

image-name: ## Print the full image name and tag.
	@echo $(IMAGE)

IMAGE_KUSTOMIZATION=config/overlays/$(OVERLAY)/kustomization.yaml
.PHONY: $(IMAGE_KUSTOMIZATION)
$(IMAGE_KUSTOMIZATION):
	mkdir -p $(dir $@)
	hack/replace-image.sh "quay.io/korrel8r/korrel8r" $(IMG) $(TAG) > $@

WATCH=kubectl get events -A --watch-only& trap "kill %%" EXIT;

# NOTE: deploy does not depend on 'image', since it may be used to deploy pre-existing images.
# To build and deploy a new image do `make image deploy`
deploy: $(IMAGE_KUSTOMIZATION)	## Deploy to a cluster using kustomize.
	$(WATCH) kubectl apply -k config/overlays/$(OVERLAY)
	$(WATCH) kubectl wait -n korrel8r --for=condition=available --timeout=60s deployment.apps/korrel8r

route:				## Create a route to access korrel8r service from outside the cluster, requires openshift.
	@oc delete -n korrel8r route/korrel8r --ignore-not-found
	@mkdir -p tmp
	@oc extract --confirm -n korrel8r configmap/openshift-service-ca.crt secret/korrel8r --to=tmp
	oc create route reencrypt -n korrel8r --service=korrel8r --cert=tmp/tls.crt --key=tmp/tls.key --dest-ca-cert=tmp/service-ca.crt --ca-cert=tmp/service-ca.crt
	$(MAKE) --no-print-directory route-url

route-url:			## Print the URL of the external route.
	@oc get -n korrel8r route/korrel8r -o template='https://{{.spec.host}}/'

# Public site is generated by .github/workflows/asciidoctor-ghpages.yml
ADOC_RUN=$(IMGTOOL) run -iq -v./docs:/src:z -v./_site:/dst:z quay.io/rhdevdocs/devspaces-documentation
ADOC_ARGS=-a revnumber=$(VERSION) -a stylesheet=fedora.css -D/dst /src/index.adoc
_site: $(wildcard docs/*.adoc) docs/zz_domains.adoc docs/zz_rest_api.adoc Makefile
	@mkdir -p $@
	$(ADOC_RUN) asciidoctor $(ADOC_ARGS)
	$(ADOC_RUN) asciidoctor-pdf -a allow-uri-read -o ebook.pdf $(ADOC_ARGS)
	@touch $@
docs/zz_domains.adoc: $(shell find cmd/korrel8r-doc internal pkg -name '*.go')
	go run ./cmd/korrel8r-doc pkg/domains/* > $@
# Note docs/templates/markdown overrides the swagger markdown templates to generate asciidoc
docs/zz_rest_api.adoc: pkg/api/zz_docs docs/templates/markdown/docs.gotmpl
	swagger -q generate markdown -T docs/templates -f $</swagger.json --output $@

release: ## Create and push a new release tag and image. Set VERSION=vX.Y.Z.
	@echo "$(VERSION)" | grep -qE "^v[0-9]+\.[0-9]+\.[0-9]+$$" || { echo "VERSION=$(VERSION) must be semantic version like vX.Y.Z"; exit 1; }
	@test -z "$(shell git status --porcelain)" || { git status -s; echo Workspace is not clean; exit 1; }
	$(MAKE) all
	hack/changelog.sh $(VERSION) > CHANGELOG.md	# Update change log
	git commit -q  -m "Release $(VERSION)" -- $(VERSION_TXT) CHANGELOG.md
	git tag $(VERSION) -a -m "Release $(VERSION)"
	$(IMGTOOL) push -q "$(IMAGE)" "$(IMG):latest"
	git push origin main --follow-tags
