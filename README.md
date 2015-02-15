> krgo was formerly [dlrootfs](https://github.com/robinmonjo/dlrootfs) and cargo but has been renamed because of this [issue]()

#krgo

docker hub without docker. `krgo` is a command line tool to pull and push docker images from/to the docker hub.
`krgo` brings the docker hub content and delivery capabilities to any container engine.

[Read the launch article and how to](https://gist.github.com/robinmonjo/f6ca0f85a204c8103e10)

##Why krgo ?

docker is really popular and a lot of people and organisations are building docker images. These images are stored
and shared on the docker hub. However they are only available to docker users. Metadata apart, a docker
image is a linux root file system that can be used with any container engine
([LXC](https://linuxcontainers.org/lxc/introduction/),
[libcontainer nsinit](https://github.com/docker/libcontainer#nsinit),
[systemd-nspawn](http://www.freedesktop.org/software/systemd/man/systemd-nspawn.html),
[rocket](https://github.com/coreos/rocket)
...).
Using `krgo`, non docker users would be able to pull and share linux images using the [docker hub](https://hub.docker.com/).

##Installation

````bash
curl -sL https://github.com/robinmonjo/krgo/releases/download/v1.5.0/krgo-v1.5.0_x86_64.tgz | tar -C /usr/local/bin -zxf -
````

Provided binary is linux only but `krgo` may be used on OSX and (probably) Windows too.

##Usage

````
NAME:
   krgo - docker hub without docker

USAGE:
   krgo [global options] command [command options] [arguments...]

VERSION:
   krgo 1.5.0 (docker 1.5.0)

COMMANDS:
   pull		pull an image
   push		push an image
   commit	commit changes to an image pulled with -g
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h		show help
   --version, -v	print the version
````

###krgo pull

`krgo pull image [-r rootfs] [-u user] [-g] [-v2]`

Pull `image` into `rootfs` directory:
- `-u` flag allows you to specify your docker hub credentials: `username:password`
- `-g` flag download the image into a git repository. Each branch contains a layer
of the image. This is the resulting rootfs of `krgo pull busybox -g`:

![Alt text](https://dl.dropboxusercontent.com/u/6543817/cargo-readme/cargo_br.png)

Branches are named `layer_<layer_index>_<layer_id>`. layer_n is a `checkout -b` from layer_n-1, so
the layer_3 branch contains the full image. You can then use it as is.

The `-g` flag brings the power of git to container images (versionning, inspecting diffs ...). But more importantly, it will allow to
push image modifications to the docker hub (see `krgo push`)

- `-v2` flag makes `krgo` download the image using docker [v2 registry](https://github.com/docker/docker-registry/issues/612). Because everything is not yet production ready, images pulled with the `-v2` flag won't be pushable to the docker hub

**Examples**:
- `krgo pull debian -v2 #library/debian:latest using v2 registry`
- `krgo pull progrium/busybox -r busybox -g`
- `krgo pull robinmonjo/debian:latest -r debian -u $DHUB_CREDS`

###krgo push

Push an image downloaded with the `-g` option to the docker hub
(a [docker hub account](https://hub.docker.com/account/signup/) is needed). Images downloaded with the `-v2` flag can't be pushed at this time as registry v2 is not yet fully operational.

In order to push your modification you **must commit** them beforehand:

`krgo commit [-r rootfs] -m "commit message"`

This will take every changes on the current branch, and commit them onto a new branch.
The new branch will be properly named and some additional metadata will be written, so
this new layer can be pushed:

````bash
$> krgo commit -m "adding new user"
Changes commited in layer_4_804c37249306321b90bbfa07d7cfe02d5f3d056971eb069d7bc37647de484a35
Image ID: 804c37249306321b90bbfa07d7cfe02d5f3d056971eb069d7bc37647de484a35
Parent: 4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8125
Layer size: 1536
Done
````

If you plan to use `krgo push`, branches should not be created manually and commit must be done via `krgo`.
Also, branches other than the last one should never be modified.

`krgo push image [-r rootfs] -u username:password`

Push the image in the `rootfs` directory onto the docker hub.

**Examples:**
- `krgo push username/debian:krgo -u $DHUB_CREDS`
- `krgo push username/busybox -r busybox -u $DHUB_CREDS`

##Dependency

If you plan to use `krgo` to push images, you will need git >= 1.8

##Notes on docker v2 registry

docker 1.5.0 pulls official images (library/*) from the v2 registry. Push are still made using the v1 registry. v2 registry brings a lot of [changes](https://github.com/docker/docker-registry/issues/612), the most noticeable ones for `krgo` are:
- images are now addressed by content (IDs are tarsum calculation)
- images are described in a manifest
- images metadata are no more stored in a json file at the root of the file system

A lot of layers in v1 where created only because the json metadata file changed. Since this file is no more distributed, some (all ?) images have "dulpicated empty layers". `krgo` clean the manifest to download only what's needed.


##Hacking on krgo

`krgo` directly uses some of docker source code. Docker is moving fast, and `krgo` must keep up.
I will maintain it but if you want to contribute every pull requests / bug reports are welcome.

You don't need linux, `krgo` can run on OSX (Windows ?). Fork the repository and clone it into your
go workspace. Then `make vendor`, `make build` and you are ready to go. Tests can be run
with `make test`. Note that most `krgo` command must be run as sudo.

##Resources

- [docker image specification](https://github.com/docker/docker/blob/master/image/spec/v1.md)
- [docker image layering](https://docs.docker.com/terms/layer/)
- [docker repository](https://github.com/docker/docker)
- [docker hub](https://hub.docker.com/)

##License

MIT
