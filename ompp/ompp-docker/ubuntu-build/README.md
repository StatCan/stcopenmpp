## stcopenmpp-build:ubuntu

### Building stcopenmpp-build:ubuntu

Clone the `stcopenmpp` repo (`git clone https://github.com/statcan/stcopenmpp.git`) and ensure the current working directory is `stcopenmpp/ompp/ompp-docker/ubuntu-build`.

In Docker Desktop's terminal (Windows PowerShell), this set of commands initiates a Ubuntu image build:

<pre>
docker build `
  --build-arg OMPP_GIT_URL=https://github.com/statcan/stcopenmpp `
  -t stcopenmpp-build:ubuntu `
  .
</pre>

The following artifacts are used in this build:

| Artifact                                                            | Version                           | URL                                                                                                 |
| :------------------------------------------------------------------ | :-------------------------------- | :-------------------------------------------------------------------------------------------------- | 
| Bison                                                               | 3.8.2                             | `https://packages.guix.gnu.org/packages/bison/3.8.2`                                                |
| curl                                                                | 8.5.0                             | `https://packages.ubuntu.com/noble/amd64/curl`                                                      |
| Flex                                                                | 2.6.4                             | `https://packages.guix.gnu.org/packages/flex/2.6.4`                                                 |
| g++                                                                 | 13.3.0                            | `https://packages.ubuntu.com/noble/amd64/gcc-13`                                                    |
| Git                                                                 | 2.43.0                            | `https://packages.ubuntu.com/noble/amd64/git`                                                       |
| Go ![Updated](https://img.shields.io/badge/Updated-green)           | 1.25.5                            | `https://pkgs.org/download/golang`                                                                  |
| Make                                                                | 4.3                               | `https://ubuntu.pkgs.org/22.04/ubuntu-main-amd64/make_4.3-4.1build1_amd64.deb.html`                 |
| Node.js ![Updated](https://img.shields.io/badge/Updated-green)      | 24.14.0                           | `https://nodejs.org/dist/v24.14.0/node-v24.14.0-linux-x64.tar.xz`                                   |
| Open MPI (libopenmpi-dev)                                           | 4.1.6-7                           | `https://ubuntu.pkgs.org/24.04/ubuntu-universe-amd64/libopenmpi-dev_4.1.6-7ubuntu2_amd64.deb.html`  |
| Open MPI (openmpi-bin)                                              | 4.1.6                             | `https://ubuntu.pkgs.org/24.04/ubuntu-universe-amd64/openmpi-bin_4.1.6-7ubuntu2_amd64.deb.html`     |
| Perl                                                                | 5.38.2                            | `https://ubuntu.pkgs.org/24.04/ubuntu-main-amd64/perl_5.38.2-3.2build2_amd64.deb.html`              |
| SQLite                                                              | 3.45.1                            | `https://packages.ubuntu.com/noble/sqlite3`                                                         |
| Ubuntu ![Updated](https://img.shields.io/badge/Updated-green)       | 24.04.3 LTS (Noble Numbat)        | `https://releases.ubuntu.com/24.04.3`                                                               |
| XZ Utils                                                            | 5.4.5                             | `https://packages.ubuntu.com/noble/xz-utils`                                                        |

### Building stcopenmpp

In Docker Desktop's terminal (Windows PowerShell), this set of commands creates and starts a container from the previously built Ubuntu image:

<pre>
docker run `
  --rm `
  -it `
  -e OMPP_BUILD_TAG=main `
  -v C:/Users/username/Downloads/ompp-build:/mnt/ompp-build `
  stcopenmpp/stcopenmpp-build:ubuntu `
  bash
</pre>

**Note:** `C:/Users/username/Downloads/ompp-build` represents the pre-existing directory on your local machine where the `stcopenmpp` build will be made available.

Once the container is running, this command initiates an `stcopenmpp` build:

<pre>
./build-all
</pre>

Once the build is finished, this command exports the build to the mounted folder:

<pre>
cp stcopenmpp-ubuntu-yyyymmdd.tar.xz /mnt/ompp-build/stcopenmpp-ubuntu-yyyymmdd.tar.xz
</pre>

Several environment variables can be set with an `-e` flag (e.g., `-e OMPP_GIT_URL=https://github.com/statcan/stcopenmpp`) when running the image:

| Name                                                                                                       | Description                                   |
| :--------------------------------------------------------------------------------------------------------- | :-------------------------------------------- |
| `OMPP_GIT_URL` Optional (default: `https://github.com/statcan/stcopenmpp`)                                 | The URL to the `stcopenmpp` repo.             |
| `OM_BUILD_CONFIGS` Optional (default: `Release,Debug`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-ubuntu-YYYYMMDD.tar.xz`)                                 | The filename of the build tarball.            |
