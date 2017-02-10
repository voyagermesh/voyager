# Build Instructions

## Requirements
- go1.5+
- glide

## Build Binary
```sh
# Install/Update dependency (needs glide)
$ glide slow

# Build
$ ./hack/make.py build voyager
```

## Build Docker
```sh
# Build Docker image
$ ./hack/docker/voyager/setup.sh
```

#### Push Docker Image
```sh
# This will push docker image to other repositories

# Add docker tag for your repository
$ docker tag appscode/voyager:<tag> <image>:<tag>

# Push Image
$ docker push <image>:<tag>

# Example:
$ docker tag appscode/voyager:default sadlil/voyager:default
$ docker push sadlil/voyager:default
```

## Build HAProxy
```sh
$ ./hack/docker/haproxy/<version>/setup.sh
```
Specific version of HAProxy can be used with Voyager via `--haproxy-image`. This packages HAProxy and [kloader](https://github.com/appscode/kloader) into a Debian jessie Docker image.
