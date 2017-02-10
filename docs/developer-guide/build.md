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
