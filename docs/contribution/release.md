# Release Process

The following steps must be done from a Linux x64 bit machine.

- Do a global replacement of tags so that docs point to the next release.
- Push changes to the release-x branch and apply new tag.
- Push all the changes to remote repo.
- Build and push voyager docker image:
```sh
$ cd ~/go/src/github.com/appscode/voyager
./hack/docker/voyager/setup.sh; env APPSCODE_ENV=prod ./hack/docker/voyager/setup.sh release
```
- Build and push haproxy image:
```sh
./hack/docker/haproxy/1.7.2/setup.sh; ./hack/docker/haproxy/1.7.2/setup.sh release
```
- Now, update the release notes in Github. See previous release notes to get an idea what to include there.


Now, you should probably also release a new version of kubed. These steps are:
- Revendor kubed so that new changes become available.
- Build kubed. Add any flags if needed.
- Push changes to release branch.
- Build and release kubed docker image.
- Now update Kubernetes salt stack files so that the new kubed image is used.
