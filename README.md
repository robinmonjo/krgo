## dlrootfs

Download rootfs from the [Docker Hub](https://registry.hub.docker.com/)

````bash
Usage: dlrootfs -i <image_name>:[<image_tag>] [-d <rootfs_destination>]

Examples:
	dlrootfs -i ubuntu  #if no tag, use latest
	dlrootfs -i ubuntu:precise
	dlrootfs -i dockefile/elasticsearch:latest
Default:
  -d="./rootfs": destination of the resulting rootfs directory
  -i="": name of the image
````

## TODO

* performance: add some concurrency
* usage demonstration (nsinit, lxc)
* tests (closely related to some docker package, need a way to find out quickly if a new docker version breaks things up)

## Warning;

* Untaring on the vagrant shared folder will fail
* cgroup-lite must be installed
