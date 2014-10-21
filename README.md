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

````bash
curl -sL https://github.com/robinmonjo/dlrootfs/releases/download/v1.3/dlrootfs_x86_64.tgz | tar -C /usr/local/bin -zxf -
````

### Why dlrootfs ?

Docker has become really popular and lots of people and organisations are building Docker images they store
and share on the [Docker Hub](https://registry.hub.docker.com/). However these images are only available for
Docker's user. `dlrootfs` allows to download root file systems from the Docker Hub so they can be used
with other container libraries/manager:

#### Using Docker images with nsinit ([`libcontainer`](https://github.com/docker/libcontainer))

1. Browse the [Docker Hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs -i ubuntu`
3. `cd` to `rootfs` and create a `container.json` file (needed by `libcontainer`, you can use the sample config of this repository `sample_configs/container.json`).
4. Launch bash in the official Docker ubuntu image: `nsinit exec /bin/bash`

#### Using Docker images with LXC

1. Browse the [Docker Hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs -i ubuntu`
3. Create a `config` file (for examples the one you can find in `sample_configs/lxc-config`)
4. Do not forget to change the `config` to match your settings (especially rootfs location)
5. Launch bash in the "official Docker ubuntu image LXC container": `lxc-start -n ubuntu -f <config file> /bin/bash`

### TODO

- [x] performance: add some concurrency
- [x] use cases (nsinit, lxc)
- [x] integration tests (closely related to some docker packages, need to find out quickly if a new Docker version breaks things up)

### Warnings

* Untaring on the `vagrant` shared folder will fail
* `cgroup-lite` is required for `nsinit`

### License

MIT
