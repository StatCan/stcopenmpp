## stcopenmpp-build:debian

### Building stcopenmpp-build:debian

Clone the `stcopenmpp` repo (`git clone https://github.com/statcan/stcopenmpp.git`) and ensure the current working directory is `stcopenmpp/ompp/ompp-docker/debian-build`.

In Docker Desktop's terminal (Windows PowerShell), this set of commands initiates a Debian image build:

<pre>
docker build `
  --build-arg OMPP_GIT_URL=https://github.com/statcan/stcopenmpp `
  -t stcopenmpp-build:debian `
  .
</pre>

The following artifacts are used in this build:

| Artifact                                                            | Version                           | URL                                                                                                 |
| :------------------------------------------------------------------ | :-------------------------------- | :-------------------------------------------------------------------------------------------------- | 
| Bison                                                               | 3.8.2                             | `https://packages.guix.gnu.org/packages/bison/3.8.2`                                                |
| curl                                                                | 8.14.1                            | `https://packages.debian.org/stable/curl`                                                           |
| Debian ![Updated](https://img.shields.io/badge/Updated-green)       | 13.2 (trixie)                     | `https://www.debian.org/releases/trixie`                                                            |
| Flex                                                                | 2.6.4                             | `https://packages.guix.gnu.org/packages/flex/2.6.4`                                                 |
| g++                                                                 | 14.2.0                            | `https://ftp.gnu.org/gnu/gcc/gcc-14.2.0`                                                            |
| Git                                                                 | 2.47.3                            | `https://debian.pkgs.org/13/debian-main-amd64/git_2.47.3-0+deb13u1_amd64.deb.html`                  |
| Go ![Updated](https://img.shields.io/badge/Updated-green)           | 1.25.5                            | `https://pkgs.org/download/golang`                                                                  |
| Make                                                                | 4.4.1                             | `https://ftp.gnu.org/gnu/make/?C=M;O=D`                                                             |
| Node.js ![Updated](https://img.shields.io/badge/Updated-green)      | 24.14.0                           | `https://nodejs.org/dist/v24.14.0/node-v24.14.0-linux-x64.tar.xz`                                   |
| Open MPI (libopenmpi-dev)                                           | 5.0.7-1                           | `https://packages.debian.org/trixie/amd64/net/libopenmpi-dev`                                       |
| Open MPI (openmpi-bin)                                              | 5.0.7                             | `https://packages.debian.org/trixie/amd64/net/openmpi-bin`                                          |
| Perl                                                                | 5.40.1                            | `https://packages.debian.org/sid/amd64/perl/download`                                               |
| SQLite                                                              | 3.46.1                            | `https://packages.debian.org/stable/sqlite3`                                                        |
| XZ Utils                                                            | 5.8.1                             | `https://packages.debian.org/stable/XZ-Utils`                                                       |

### Building stcopenmpp

In Docker Desktop's terminal (Windows PowerShell), this set of commands creates and starts a container from the previously built Debian image:

<pre>
docker run `
  --rm `
  -it `
  -e OMPP_BUILD_TAG=main `
  -v C:/Users/username/Downloads/ompp-build:/mnt/ompp-build `
  stcopenmpp/stcopenmpp-build:debian `
  bash
</pre>

**Note:** `C:/Users/username/Downloads/ompp-build` represents the pre-existing directory on your local machine where the `stcopenmpp` build will be made available.

Once the container is running, this command initiates an `stcopenmpp` build:

<pre>
./build-all
</pre>

Once the build is finished, this command exports the build to the mounted folder:

<pre>
cp stcopenmpp-debian-yyyymmdd.tar.xz /mnt/ompp-build/stcopenmpp-debian-yyyymmdd.tar.xz
</pre>

Several environment variables can be set with an `-e` flag (e.g., `-e OMPP_GIT_URL=https://github.com/statcan/stcopenmpp`) when running the image:

| Name                                                                                                       | Description                                   |
| :--------------------------------------------------------------------------------------------------------- | :-------------------------------------------- |
| `OMPP_GIT_URL` Optional (default: `https://github.com/statcan/stcopenmpp`)                                 | The URL to the `stcopenmpp` repo.             |
| `OM_BUILD_CONFIGS` Optional (default: `Release,Debug`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-debian-YYYYMMDD.tar.xz`)                                 | The filename of the build tarball.            |
