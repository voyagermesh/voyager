# make build-local //build for local go env configs
# make build-local GOOS= GOARCH= GOARM= //override default values
# make build-docker //build inside docker, for local go env configs
# make build-docker GOOS= GOARCH= GOARM= //build inside docker, override local configs
# make docker-build //build docker image for local go env configs
# make docker-build GOOS= GOARCH= GOARM= IMAGE_NAME=(voyager if unspecified) IMAGE_TYPE=(debug if unspecified)
# make docker-push
# make docker-push GOOS= GOARCH= GOARM=
# make docker-release
# make docker-release GOOS= GOARCH= GOARM=
# make package-deb //you can also specify GOOS GOARCH GOARM values
# make package-rpm //same as above
# make version
# make metadata
# make fmt
# make test-unit ARGS="....."
# make test-minikube ARGS="....."
# make test-e2e ARGS="....."

test-unit:
	go install ./...
	go test -v . ./api/... ./client/... ./pkg/... $(ARGS)

test-minikube:
	go install ./...
	go test -v ./test/e2e/... -timeout 10h -args -ginkgo.v -ginkgo.progress -ginkgo.trace -v=2 -cloud-provider=minikube $(ARGS)

test-e2e:
	go install ./...
	go test -v ./test/e2e/... -timeout 10h -args -ginkgo.v -ginkgo.progress -ginkgo.trace -v=2 $(ARGS)

include Makefile.metadata

haproxy_dockerfile_dir=hack/docker/haproxy/$(haproxy_version)-alpine
haproxy_image_tag=$(haproxy_version)-$(version)-alpine
voyager_dockerfile_dir=hack/docker/voyager
voyager_image_tag=$(version)

docker-build-%-debug:
	@$(MAKE) --no-print-directory IMAGE_NAME=$* IMAGE_TYPE=debug docker-build

docker-build-%-prod:
	@$(MAKE) --no-print-directory IMAGE_NAME=$* IMAGE_TYPE=prod docker-build

docker-build: build-docker
	@cowsay -f tux building $(DOCKER_REGISTRY)/$(IMAGE_NAME)-$(GOOS)-$(GOARCH)$(GOARM):$($(IMAGE_NAME)_image_tag)-$(IMAGE_TYPE)

	env IMAGE_TYPE=$(IMAGE_TYPE) \
		IMAGE_NAME=$(IMAGE_NAME) \
		DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
		GOOS=$(GOOS) \
		GOARCH=$(GOARCH) \
		GOARM=$(GOARM) \
		TAG=$($(IMAGE_NAME)_image_tag) \
		DOCKERFILE_DIR=$($(IMAGE_NAME)_dockerfile_dir) \
		./$($(IMAGE_NAME)_dockerfile_dir)/make.sh

docker-push-%-debug:
	@$(MAKE) --no-print-directory IMAGE_NAME=$* IMAGE_TYPE=debug docker-push

docker-push-%-prod:
	@$(MAKE) --no-print-directory IMAGE_NAME=$* IMAGE_TYPE=debug docker-push

docker-push: docker-build
	@cowsay -f tux pushing $(DOCKER_REGISTRY)/$(IMAGE_NAME):$($(IMAGE_NAME)_image_tag)-$(GOOS)-$(GOARCH)$(GOARM)-$(IMAGE_TYPE)
	@if [ "$$APPSCODE_ENV" = "prod" ]; then\
		echo "Nothing to do in prod env. Are you trying to 'release' binaries to prod?";\
		exit 1;\
	fi
	@if [ "$(version_strategy)" = "git_tag" ]; then\
		echo "Are you trying to 'release' binaries to prod?";\
		exit 1;\
	fi

	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$($(IMAGE_NAME)_image_tag)-$(GOOS)-$(GOARCH)$(GOARM)-$(IMAGE_TYPE)

	@if [[ "$(version_strategy)" == "commit_hash" && "$(git_branch)" == "master" ]]; then\
		set -x;\
		docker tag $(DOCKER_REGISTRY)/$(image_name):$(image_tag) $(DOCKER_REGISTRY)/$(image_name):canary ;\
		docker push $(DOCKER_REGISTRY)/$(image_name):canary ;\
	fi

docker-release: docker-build
	@if [ "$$APPSCODE_ENV" != "prod" ]; then\
		echo "'release' only works in PROD env.";\
		exit 1;\
	fi

	@if [ "$(version_strategy)" != "git_tag" ]; then\
		echo "'apply_tag' to release binaries and/or docker images.";\
		exit 1;\
	fi

	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$($(IMAGE_NAME)_image_tag)-$(GOOS)-$(GOARCH)$(GOARM)-$(IMAGE_TYPE)

package-%:
	@mkdir -p dist/$(BIN)/package
	$(MAKE) --no-print-directory dist/$(BIN)/package/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM).$*

gen:
	./hack/gen.sh

fmt:
	gofmt -s -w *.go apis client pkg test third_party
	goimports -w *.go apis client pkg test third_party


SOURCES := $(shell find . -name "*.go")
build-local:
	@$(MAKE) --no-print-directory dist/$(BIN)/local/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM)$(EXT)
build-docker:
	@$(MAKE) --no-print-directory dist/$(BIN)/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM)$(EXT)

# build locally
dist/$(BIN)/local/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM)$(EXT): $(SOURCES)
	@cowsay -f tux building binary $(BIN)-$(GOOS)-$(GOARCH)$(GOARM)$(EXT)
	@$(MAKE) gen
	@$(MAKE) fmt
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(CGO_ENV) \
		go build -o dist/$(BIN)/local/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM) \
		$(CGO) $(ldflags) *.go

# build inside docker
dist/$(BIN)/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM)$(EXT): $(SOURCES)
	@cowsay -f tux building binary $(BIN)-$(GOOS)-$(GOARCH)$(GOARM)$(EXT) inside docker
	@$(MAKE) gen
	@$(MAKE) fmt

	docker run --rm -u $(UID) -v /tmp:/.cache -v $$(pwd):/go/src/$(PKG) -w /go/src/$(PKG) \
		-e $(CGO_ENV) golang:1.10.0 env GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(CGO_ENV) \
		go build -o dist/$(BIN)/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM) $(CGO) $(ldflags) *.go

# nfpm
dist/$(BIN)/package/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM).%:
	@cowsay -f tux creating package $(BIN)-$(GOOS)-$(GOARCH)$(GOARM).$*
	docker run --rm -v $(REPO_ROOT):/go/src/$(PKG) -w /go/src/$(PKG) tahsin/releaser:latest /bin/bash -c \
		"sed -i 's/amd64/$(GOARCH)$(GOARM)/' /nfpm.yaml; \
		sed -i 's/linux/$(GOOS)/' /nfpm.yaml; \
		sed -i '4s/1.0.0/$(version)/' /nfpm.yaml; \
		nfpm pkg --target /go/src/$(PKG)/dist/$(BIN)/package/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM).$* -f /nfpm.yaml"

# compress
compress: build-docker
	@cowsay -f tux compressing $(BIN)-$(GOOS)-$(GOARCH)$(GOARM)
	if [ $(GOOS) != windows ]; then \
		upx --brute dist/$(BIN)/$(BIN)-$(GOOS)-$(GOARCH)$(GOARM); \
	fi
