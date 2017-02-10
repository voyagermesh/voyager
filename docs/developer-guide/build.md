# Build Instructions

## Requirements
- go1.5+
- glide

## Build Binary
```sh
# Install/Update dependency (needs glide)
$ glide slow

# Build
$ ./hack/make.py build Voyager
```

## Build Docker
```sh
# Build Docker image
$ ./hack/docker/Voyager/setup.sh
```

#### Push Docker Image
```sh
# This will push docker image to other repositories

# Add docker tag for your repository
$ docker tag appscode/Voyager:<tag> <image>:<tag>

# Push Image
$ docker push <image>:<tag>

# Example:
$ docker tag appscode/Voyager:default sadlil/Voyager:default
$ docker push sadlil/Voyager:default
```

## Build HAProxy
```sh
$ ./hack/docker/haproxy/<version>/setup.sh
```
Specific version of HAProxy can be used with Voyager controller via `--haproxy-image`. This build is depending upon [kloader](https://github.com/appscode/kloader).