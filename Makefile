GIT_COMMIT=$(shell git rev-list -1 HEAD)
GIT_TAG=$(shell git tag --sort="-version:refname" | head -n 1)
all:
	go build -ldflags "-X main.GitCommit=${GIT_COMMIT} -X main.GitTag=${GIT_TAG}"


