For more details about `stcopenmpp`, visit <a href="https://github.com/statcan/stcopenmpp" target="new">github.com/statcan/stcopenmpp</a>.

<br>

# Contents
* <a href="#windows">Windows</a>
* <a href="#linux-debian">Linux (Debian)</a>
* <a href="#linux-ubuntu">Linux (Ubuntu)</a>

<br>

# Windows

### Packages

The following packages are used in this build:

| Package                                                               | Version                           | URL                                                                                                                                   |
| :-------------------------------------------------------------------- | :-------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------ | 
| 7-Zip ![Updated](https://img.shields.io/badge/Updated-green)          | 26.04                             | `https://www.7-zip.org/a/7z2604-x64.exe`                                                                                              |
| Git ![Updated](https://img.shields.io/badge/Updated-green)            | 2.55.0                            | `https://github.com/git-for-windows/git/releases/download/v2.55.0.windows.3/Git-2.55.0.3-64-bit.exe`                                  |
| Go ![Updated](https://img.shields.io/badge/Updated-green)             | 1.26.5                            | `https://go.dev/dl/go1.26.5.windows-amd64.zip`                                                                                        |
| Microsoft MPI                                                         | 10.1.12498.52                     | `https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisetup.exe`                                   |
| Microsoft MPI SDK                                                     | 10.1.12498.52                     | `https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisdk.msi`                                     |
| Microsoft Visual Studio Build Tools                                   | 17.4.35026.314                    | `https://aka.ms/vs/17/release.ltsc.17.4/vs_buildtools.exe`                                                                            |
| MinGW                                                                 | 20.0                              | `https://nuwen.net/files/mingw/history/mingw-20.0.exe`                                                                                |
| Node.js ![Updated](https://img.shields.io/badge/Updated-green)        | 24.20.0                           | `https://nodejs.org/dist/v24.20.0/node-v24.20.0-win-x64.zip`                                                                          |
| Perl (x64) ![Updated](https://img.shields.io/badge/Updated-green)     | 5.42.2.1                          | `https://github.com/StrawberryPerl/Perl-Dist-Strawberry/releases/download/SP_54221_64bit/strawberry-perl-5.42.2.1-64bit-portable.zip` |
| Perl (x86)                                                            | 5.32.1.1                          | `https://strawberryperl.com/download/5.32.1.1/strawberry-perl-no64-5.32.1.1-32bit-portable.zip`                                       |
| SQLite ![Updated](https://img.shields.io/badge/Updated-green)         | 3.53.3                            | `https://sqlite.org/2026/sqlite-tools-win-x64-3530300.zip`                                                                            |
| Windows Server ![Updated](https://img.shields.io/badge/Updated-green) | ltsc2025                          | `https://hub.docker.com/r/microsoft/windows-servercore`                                                                               |
| winflexbison                                                          | 2.5.25                            | `https://github.com/lexxmark/winflexbison/releases/download/v2.5.25/win_flex_bison-2.5.25.zip`                                        |

### Download build image

<pre>docker pull stcopenmpp/stcopenmpp-build:windows</pre>

### Create container based on build image

In Docker Desktop's terminal (Windows PowerShell), this set of commands creates and starts a container from the previously downloaded image:

<pre>docker run `
  --rm `
  -it `
  -e OMPP_BUILD_TAG=main `
  -v c:\users\username\downloads\ompp-build:c:\build `
  stcopenmpp/stcopenmpp-build:windows
</pre>

**Note:** `c:\users\username\downloads\ompp-build` represents the pre-existing directory on your local machine where the stcopenmpp build will be made available.

The image's default `ENTRYPOINT` is Windows CMD.

### Build stcopenmpp inside container

Once the container is running, this command initiates an `stcopenmpp` build:

<pre>build-all.bat</pre>

Several environment variables can be set with an `-e` flag (e.g., `-e OMPP_GIT_URL=https://github.com/statcan/stcopenmpp`) when running the image:

| Name                                                                                                       | Description                                   |
| :--------------------------------------------------------------------------------------------------------- | :-------------------------------------------- |
| `OMPP_GIT_URL` Optional (default: `https://github.com/statcan/stcopenmpp`)                                 | The URL to the `stcopenmpp` repo.             |
| `OM_BUILD_CONFIGS` Optional (default: `Release,Debug`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-windows-YYYYMMDD.zip`)                                   | The filename of the build zip.                |

<br>

# Linux (Debian)

### Packages

The following packages are used in this build:

| Package                                                             | Version                           | URL                                                                                                 |
| :------------------------------------------------------------------ | :-------------------------------- | :-------------------------------------------------------------------------------------------------- | 
| Bison                                                               | 3.8.2                             | `https://packages.debian.org/trixie/bison`                                                          |
| curl                                                                | 8.14.1                            | `https://packages.debian.org/trixie/curl`                                                           |
| Debian ![Updated](https://img.shields.io/badge/Updated-green)       | 13.7 (trixie)                     | `https://www.debian.org/releases/trixie`                                                            |
| Flex                                                                | 2.6.4                             | `https://packages.debian.org/trixie/flex`                                                           |
| g++                                                                 | 14.2.0                            | `https://packages.debian.org/trixie/g++`                                                            |
| Git                                                                 | 2.47.3                            | `https://packages.debian.org/trixie/git`                                                            |
| Go ![Updated](https://img.shields.io/badge/Updated-green)           | 1.26.8                            | `https://pkgs.org/download/golang`                                                                  |
| Make                                                                | 4.4.1                             | `https://packages.debian.org/trixie/make`                                                           |
| Node.js ![Updated](https://img.shields.io/badge/Updated-green)      | 24.18.0                           | `https://nodejs.org/dist/v24.18.0/node-v24.18.0-linux-x64.tar.xz`                                   |
| Open MPI (libopenmpi-dev)                                           | 5.0.7                             | `https://packages.debian.org/trixie/libopenmpi-dev`                                                 |
| Open MPI (openmpi-bin)                                              | 5.0.7                             | `https://packages.debian.org/trixie/openmpi-bin`                                                    |
| Perl                                                                | 5.40.1                            | `https://packages.debian.org/trixie/perl`                                                           |
| SQLite                                                              | 3.46.1                            | `https://packages.debian.org/trixie/sqlite3`                                                        |
| XZ Utils                                                            | 5.8.1                             | `https://packages.debian.org/trixie/xz-utils`                                                       |

### Download build image

<pre>docker pull stcopenmpp/stcopenmpp-build:debian</pre>

### Create container based on build image

In Docker Desktop's terminal (Windows PowerShell), this set of commands creates and starts a container from the previously downloaded image:

<pre>docker run `
  --rm `
  -it `
  -e OMPP_BUILD_TAG=main `
  -v c:\users\username\downloads\ompp-build:c:\build `
  stcopenmpp/stcopenmpp-build:debian
</pre>

**Note:** `c:\users\username\downloads\ompp-build` represents the pre-existing directory on your local machine where the stcopenmpp build will be made available.

### Build stcopenmpp inside container

Once the container is running, this command initiates an `stcopenmpp` build:

<pre>./build-all</pre>

Once the build is finished, this command exports the build to the mounted folder:

<pre>cp stcopenmpp-debian-yyyymmdd.tar.xz /mnt/ompp-build/stcopenmpp-debian-yyyymmdd.tar.xz</pre>

Several environment variables can be set with an `-e` flag (e.g., `-e OMPP_GIT_URL=https://github.com/statcan/stcopenmpp`) when running the image:

| Name                                                                                                       | Description                                   |
| :--------------------------------------------------------------------------------------------------------- | :-------------------------------------------- |
| `OMPP_GIT_URL` Optional (default: `https://github.com/statcan/stcopenmpp`)                                 | The URL to the `stcopenmpp` repo.             |
| `OM_BUILD_CONFIGS` Optional (default: `RELEASE,DEBUG`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-debian-YYYYMMDD.xz`)                                     | The filename of the build tarball.            |

<br>

# Linux (Ubuntu)

### Packages

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

### Download build image

<pre>docker pull stcopenmpp/stcopenmpp-build:ubuntu</pre>

### Create container based on build image

In Docker Desktop's terminal (Windows PowerShell), this set of commands creates and starts a container from the previously downloaded image:

<pre>docker run `
  --rm `
  -it `
  -e OMPP_BUILD_TAG=main `
  -v c:\users\username\downloads\ompp-build:c:\build `
  stcopenmpp/stcopenmpp-build:ubuntu
</pre>

**Note:** `c:\users\username\downloads\ompp-build` represents the pre-existing directory on your local machine where the stcopenmpp build will be made available.

### Build stcopenmpp inside container

Once the container is running, this command initiates an `stcopenmpp` build:

<pre>./build-all</pre>

Once the build is finished, this command exports the build to the mounted folder:

<pre>cp stcopenmpp-ubuntu-yyyymmdd.tar.xz /mnt/ompp-build/stcopenmpp-ubuntu-yyyymmdd.tar.xz</pre>

Several environment variables can be set with an `-e` flag (e.g., `-e OMPP_GIT_URL=https://github.com/statcan/stcopenmpp`) when running the image:

| Name                                                                                                       | Description                                   |
| :--------------------------------------------------------------------------------------------------------- | :-------------------------------------------- |
| `OMPP_GIT_URL` Optional (default: `https://github.com/statcan/stcopenmpp`)                                 | The URL to the `stcopenmpp` repo.             |
| `OM_BUILD_CONFIGS` Optional (default: `RELEASE,DEBUG`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-ubuntu-YYYYMMDD.xz`)                                     | The filename of the build tarball.            |