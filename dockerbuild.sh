export DOCKER_BUILDKIT=1
docker build -t inkstone --no-cache --build-arg VERSION="$(git describe --tags --always)" .
