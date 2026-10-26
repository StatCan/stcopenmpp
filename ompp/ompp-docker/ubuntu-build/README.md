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

The following packages are used in this build:

| Package                                                                           | Version                           | URL                                                                                                 |
| :-------------------------------------------------------------------------------- | :-------------------------------- | :-------------------------------------------------------------------------------------------------- | 
| Bison                                                                             | 3.8.2                             | `https://packages.ubuntu.com/eu/resolute/amd64/bison`                                               |
| curl ![Updated](https://img.shields.io/badge/Updated-green)                       | 8.18.0                            | `https://packages.ubuntu.com/eu/resolute/amd64/curl`                                                |
| Flex                                                                              | 2.6.4                             | `https://packages.ubuntu.com/eu/resolute/amd64/flex`                                                |
| g++ ![Updated](https://img.shields.io/badge/Updated-green)                        | 15.2.0                            | `https://packages.ubuntu.com/eu/resolute/amd64/g++`                                                 |
| Git ![Updated](https://img.shields.io/badge/Updated-green)                        | 2.53.0                            | `https://packages.ubuntu.com/eu/resolute/amd64/git`                                                 |
| Go ![Updated](https://img.shields.io/badge/Updated-green)                         | 1.26.8                            | `https://pkgs.org/download/golang`                                                                  |
| Make ![Updated](https://img.shields.io/badge/Updated-green)                       | 4.4.1                             | `https://packages.ubuntu.com/eu/resolute/amd64/make`                                                |
| Node.js ![Updated](https://img.shields.io/badge/Updated-green)                    | 24.20.0                           | `https://nodejs.org/dist/v24.20.0/node-v24.20.0-linux-x64.tar.xz`                                   |
| Open MPI (libopenmpi-dev) ![Updated](https://img.shields.io/badge/Updated-green)  | 5.0.10                            | `https://packages.ubuntu.com/eu/resolute/amd64/libopenmpi-dev`                                      |
| Open MPI (openmpi-bin) ![Updated](https://img.shields.io/badge/Updated-green)     | 5.0.10                            | `https://packages.ubuntu.com/eu/resolute/amd64/openmpi-bin`                                         |
| Perl ![Updated](https://img.shields.io/badge/Updated-green)                       | 5.40.1                            | `https://packages.ubuntu.com/eu/resolute/amd64/perl`                                                |
| SQLite ![Updated](https://img.shields.io/badge/Updated-green)                     | 3.46.1                            | `https://packages.ubuntu.com/eu/resolute/amd64/sqlite3`                                             |
| Ubuntu ![Updated](https://img.shields.io/badge/Updated-green)                     | 26.04 LTS (Resolute Raccoon)      | `https://releases.ubuntu.com/26.04/`                                                                |
| XZ Utils ![Updated](https://img.shields.io/badge/Updated-green)                   | 5.8.3                             | `https://packages.ubuntu.com/eu/resolute/amd64/xz-utils                                             |

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

| Name                                                                                                       | Description                                    |
| :--------------------------------------------------------------------------------------------------------- | :--------------------------------------------- |
| `OMPP_GIT_URL` Optional (default: `https://github.com/statcan/stcopenmpp`)                                 | The URL to the `stcopenmpp` repo.              |
| `OM_BUILD_CONFIGS` Optional (default: `RELEASE,DEBUG`)                                                     | The build(s) to make.                          |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                        |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built.  |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.       |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-debian-YYYYMMDD.tar.xz`)                                 | The filename of the build tarball.             |
