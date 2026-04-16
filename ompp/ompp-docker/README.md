For more details about `stcopenmpp`, visit <a href="https://github.com/statcan/stcopenmpp" target="new">github.com/statcan/stcopenmpp</a>.

<br>

# Contents
* <a href="#windows">Windows</a>
* <a href="#linux-debian">Linux (Debian)</a>
* <a href="#linux-ubuntu">Linux (Ubuntu)</a>

<br>

# Windows

### Artifacts

The following artifacts are used in this build:

| Artifact                                                            | Version                           | URL                                                                                                 |
| :------------------------------------------------------------------ | :-------------------------------- | :-------------------------------------------------------------------------------------------------- | 
| 7-Zip                                                               | 25.01                             | `https://www.7-zip.org/a/7z2501-x64.exe`                                                            |
| Git ![Updated](https://img.shields.io/badge/Updated-green)          | 2.52.0                            | `https://github.com/git-for-windows/git/releases/download/v2.52.0.windows.1/Git-2.52.0-64-bit.exe`  |
| Go ![Updated](https://img.shields.io/badge/Updated-green)           | 1.25.5                            | `https://go.dev/dl/go1.25.5.windows-amd64.zip`                                                      |
| Microsoft MPI                                                       | 10.1.12498.52                     | `https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisetup.exe` |
| Microsoft MPI SDK                                                   | 10.1.12498.52                     | `https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisdk.msi`   |
| Microsoft Visual Studio Build Tools                                 | 17.4.35026.314                    | `https://aka.ms/vs/17/release.ltsc.17.4/vs_buildtools.exe`                                          |
| MinGW ![Updated](https://img.shields.io/badge/Updated-green)        | 20.0                              | `https://nuwen.net/files/mingw/history/mingw-20.0.exe`                                                      |
| Node.js ![Updated](https://img.shields.io/badge/Updated-green)      | 24.14.0                           | `https://nodejs.org/dist/v24.14.0/node-v24.14.0-win-x64.zip`                                        |
| Perl (x64)                                                          | 5.32.1.1                          | `https://strawberryperl.com/download/5.32.1.1/strawberry-perl-5.32.1.1-64bit-portable.zip`          |
| Perl (x86)                                                          | 5.32.1.1                          | `https://strawberryperl.com/download/5.32.1.1/strawberry-perl-no64-5.32.1.1-32bit-portable.zip`     |
| SQLite ![Updated](https://img.shields.io/badge/Updated-green)       | 3.51.0                            | `https://www.sqlite.org/2025/sqlite-tools-win-x64-3510100.zip`                                      |
| winflexbison                                                        | 2.5.25                            | `https://github.com/lexxmark/winflexbison/releases/download/v2.5.25/win_flex_bison-2.5.25.zip`      |

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

### Artifacts

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
| `OM_BUILD_CONFIGS` Optional (default: `Release,Debug`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-debian-YYYYMMDD.xz`)                                   | The filename of the build tarball.                |

<br>

# Linux (Ubuntu)

### Artifacts

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
| `OM_BUILD_CONFIGS` Optional (default: `Release,Debug`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-ubuntu-YYYYMMDD.xz`)                                   | The filename of the build tarball.                |