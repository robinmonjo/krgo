## dlrootfs

Download root file systems from the [Docker Hub](https://registry.hub.docker.com/)

````bash
Usage: dlrootfs -i <image_name>[:<image_tag>] [-d <rootfs_destination>]

Examples:
	dlrootfs -i ubuntu  #if no tag, use latest
	dlrootfs -i ubuntu:precise
	dlrootfs -i dockefile/elasticsearch:latest
Default:
  -d="./rootfs": destination of the resulting rootfs directory
  -i="": name of the image
````

### Installation
 TODO

### Why dlrootfs ?

Docker has become really popular and lots of people and organisations are building Docker images they store
and share on the [Docker Hub](https://registry.hub.docker.com/). However these images are only available for
Docker's user. `dlrootfs` allows to download root file systems from the Docker Hub so they can be used
with other container libraries/manager:

#### Using Docker images with nsinit ([`libcontainer`](https://github.com/docker/libcontainer))

1. Browse the [Docker Hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs -i ubuntu`
3. `cd` to `rootfs` and create a `container.json` file (needed by `libcontainer`, you can use the sample config of this repository `misc/container.json`).
4. Launch bash in the official Docker ubuntu image: `nsinit exec /bin/bash`

#### Using Docker images with LXC

1. Browse the [Docker Hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs -i ubuntu`
3. Create a `fstab` and a `config` file (for examples the one you can find in `misc`)
4. Do not forget to change the `config` to match your settings
5. Launch bash in the "official Docker ubuntu image LXC container": `lxc-start -n ubuntu -f <config file> /bin/bash`

### TODO

- [ ] performance: add some concurrency
- [x] use cases (nsinit, lxc)
- [ ] integration tests (closely related to some docker packages, need to find out quickly if a new Docker version breaks things up)

### Warnings

* Untaring on the `vagrant` shared folder will fail
* `cgroup-lite` is required for `nsinit`

### License

MIT
