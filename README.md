## dlrootfs

Download root file systems from the [docker hub](https://registry.hub.docker.com/) without docker

````bash
Usage: dlrootfs <image_name>:[<image_tag>] [-d <rootfs_destination>] [-u <username>:<password>] [-g]

Examples:
  dlrootfs ubuntu  #if no tag, use latest
  dlrootfs ubuntu:precise -d ubuntu_rootfs
  dlrootfs dockefile/elasticsearch:latest
  dlrootfs my_repo/my_image:latest -u username:password
  dlrootfs version
Default:
  -d="./rootfs": destination of the resulting rootfs directory
  -g=false: use git layering
  -u="": docker hub credentials: <username>:<password>
````
#### `-g` option

As explained [in the doc](https://docs.docker.com/terms/layer/), docker images are a set of layers. Using the `-g` option,
`dlrootfs` will download the file system in a git repository where each layer is downloaded in a separate branch:

![Alt text](https://dl.dropboxusercontent.com/u/6543817/dlrootfs-readme/dlrootfs-g.png)

The screenshot above is the resulting rootfs of `dlrootfs ubuntu -g`. We can clearly see the image is composed of 5 layers.
`layer(n)_*` result from `git checkout -b layer(n-1)_*` with data from `layer(n)`.

It allows to use git to see diffs between layers, checkout a new branch, work on the rootfs with a container engine, review
and commit changes, etc. It also opens the path for `docker push` without docker (coming soon).

### Installation

````bash
curl -sL https://github.com/robinmonjo/dlrootfs/releases/download/v1.4.0/dlrootfs_x86_64.tgz | tar -C /usr/local/bin -zxf -
````

Provided binary is Linux only but `dlrootfs` may be used on OSX and (probably) windows too.
The difference is, when ran on a linux box, `dlrootfs` will perform `lchown` during layer extraction,
it won't otherwise.

Some images require you to be root during extraction (the official busybox image for example) why others won't
(the official debian one).

### Why dlrootfs ?

Docker has become really popular and lots of people and organisations are building Docker images they store
and share on the [docker hub](https://registry.hub.docker.com/). However these images are only available for
Docker's user. `dlrootfs` allows to download root file systems from the Docker Hub so they can be used
with other container engines ([LXC](https://linuxcontainers.org/), [nsinit (`libcontainer`)](https://github.com/docker/libcontainer), [systemd-nspawn](http://0pointer.de/public/systemd-man/systemd-nspawn.html) ...)


##### Using docker images with nsinit

1. Browse the [docker hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs ubuntu`
3. `cd` to `rootfs` and create a `container.json` file (needed by `libcontainer`, you can use the sample config of this repository `sample_configs/container.json`).
4. Launch bash in the official Docker ubuntu image: `nsinit exec /bin/bash`

##### Using docker images with LXC

1. Browse the [docker hub](https://registry.hub.docker.com/) and find the image you want (say [ubuntu](https://registry.hub.docker.com/u/library/ubuntu/))
2. Download ubuntu rootfs: `dlrootfs ubuntu`
3. Create a `config` file (for examples the one you can find in `sample_configs/lxc-config`)
4. Do not forget to change the `config` to match your settings (especially rootfs location)
5. Launch bash in the "official Docker ubuntu image LXC container": `lxc-start -n ubuntu -f <config file> /bin/bash`

### Warnings

* Untaring on the `vagrant` shared folder will fail
* `cgroup-lite` is required for `nsinit`

### License

MIT
