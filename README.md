## dlrootfs

Download root file systems from the [Docker Hub](https://registry.hub.docker.com/)

````bash
Usage: dlrootfs -i <image_name>:[<image_tag>] [-d <rootfs_destination>] [-u <username>:<password>]

Examples:
  dlrootfs -i ubuntu  #if no tag, use latest
  dlrootfs -i ubuntu:precise -d ubuntu_rootfs
  dlrootfs -i dockefile/elasticsearch:latest
  dlrootfs -i my_repo/my_image:latest -u username:password
Default:
  -d="./rootfs": destination of the resulting rootfs directory
  -i="": name of the image <repository>/<image>:<tag>
  -u="": docker hub credentials: <username>:<password>
  -v=false: display dlrootfs version
````

### Installation

````bash
curl -sL https://github.com/robinmonjo/dlrootfs/releases/download/v1.3/dlrootfs_x86_64.tgz | tar -C /usr/local/bin -zxf -
````

### Why dlrootfs ?

Docker has become really popular and lots of people and organisations are building Docker images they store
and share on the [Docker Hub](https://registry.hub.docker.com/). However these images are only available for
Docker's user. `dlrootfs` allows to download root file systems from the Docker Hub so they can be used
with other container engines ([LXC](https://linuxcontainers.org/), [nsinit (`libcontainer`)](https://github.com/docker/libcontainer), [systemd-nspawn](http://0pointer.de/public/systemd-man/systemd-nspawn.html) ...)


##### Using Docker images with nsinit

1. Browse the [Docker Hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs -i ubuntu`
3. `cd` to `rootfs` and create a `container.json` file (needed by `libcontainer`, you can use the sample config of this repository `sample_configs/container.json`).
4. Launch bash in the official Docker ubuntu image: `nsinit exec /bin/bash`

##### Using Docker images with LXC

1. Browse the [Docker Hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs -i ubuntu`
3. Create a `config` file (for examples the one you can find in `sample_configs/lxc-config`)
4. Do not forget to change the `config` to match your settings (especially rootfs location)
5. Launch bash in the "official Docker ubuntu image LXC container": `lxc-start -n ubuntu -f <config file> /bin/bash`

### Notes

Provided binary is Linux only but `dlrootfs` may be used on OSX and (probably) windows too.
The difference is, when ran on a Linux box, `dlrootfs` will perform `lchown` during layer extraction,
it won't otherwise.

Some images require you to be root during extraction (the official busybox image for example) why others won't
(the official debian one). Not sure exactly why but probably because of the way they were packaged.


### Warnings

* Untaring on the `vagrant` shared folder will fail
* `cgroup-lite` is required for `nsinit`

### License

MIT
