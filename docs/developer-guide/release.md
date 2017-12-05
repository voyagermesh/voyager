---
title: Release | Voyager
description: Voyager Release
menu:
  product_voyager_5.0.0-rc.6:
    identifier: release    
    name: Release
    parent: developer-guide
    weight: 15
product_name: voyager
menu_name: product_voyager_5.0.0-rc.6
section_menu_id: developer-guide
---

# Release Process

The following steps must be done from a Linux x64 bit machine.

- Do a global replacement of tags so that docs point to the next release.
- Push changes to the release-x branch and apply new tag.
- Push all the changes to remote repo.
- Build and push voyager docker image:
```console
$ cd ~/go/src/github.com/appscode/voyager
./hack/docker/voyager/setup.sh; env APPSCODE_ENV=prod ./hack/docker/voyager/setup.sh release
```
- Build and push haproxy image:
```console
./hack/docker/haproxy/1.7.6/setup.sh; ./hack/docker/haproxy/1.7.6/setup.sh release
```
Note that, HAProxy image bundles [kloader](https://github.com/appscode/kloader). If you need to update kloader version, modify the setup.sh file for HAProxy. See [here](/hack/docker/haproxy/1.7.5/setup.sh#L20) for an example.

- Now, update the release notes in Github. See previous release notes to get an idea what to include there.
