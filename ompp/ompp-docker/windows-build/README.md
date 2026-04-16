## stcopenmpp-build:windows

### Building stcopenmpp-build:windows

Clone the `stcopenmpp` repo (`git clone https://github.com/statcan/stcopenmpp.git`) and ensure the current working directory is `stcopenmpp/ompp/ompp-docker/windows-build`.

In Docker Desktop's terminal (Windows PowerShell), this set of commands initiates a Windows image build:

<pre>
docker build `
  --build-arg OMPP_GIT_URL=https://github.com/statcan/stcopenmpp `
  -t stcopenmpp-build:windows `
  .
</pre>

The following artifacts are used in this build:

| Artifact                                                            | Version                           | URL                                                                                                 |
| :------------------------------------------------------------------ | :-------------------------------- | :-------------------------------------------------------------------------------------------------- | 
| 7-Zip                                                               | 25.01                             | `https://www.7-zip.org/a/7z2501-x64.exe`                                                            |
| Git ![Updated](https://img.shields.io/badge/Updated-green)          | 2.52.0                            | `https://github.com/git-for-windows/git/releases/download/v2.52.0.windows.1/Git-2.52.0-64-bit.exe`  |
| Go ![Updated](https://img.shields.io/badge/Updated-green)           | 1.25.5                            | `https://go.dev/dl/go1.25.5.windows-amd64.zip`                                                      |
| Microsoft MPI                                                       | 10.1.12498.52                     | `https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisetup.exe` |
| Microsoft MPI SDK                                                   | 10.1.12498.52                     | `https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisdk.msi`   |
| Microsoft Visual Studio Build Tools                                 | 17.4.35026.314                    | `https://aka.ms/vs/17/release.ltsc.17.4/vs_buildtools.exe`                                          |
| MinGW ![Updated](https://img.shields.io/badge/Updated-green)        | 20.0                              | `https://nuwen.net/files/mingw/history/mingw-20.0.exe`                                              |
| Node.js ![Updated](https://img.shields.io/badge/Updated-green)      | 24.14.0                           | `https://nodejs.org/dist/v24.14.0/node-v24.14.0-win-x64.zip`                                        |
| Perl (x64)                                                          | 5.32.1.1                          | `https://strawberryperl.com/download/5.32.1.1/strawberry-perl-5.32.1.1-64bit-portable.zip`          |
| Perl (x86)                                                          | 5.32.1.1                          | `https://strawberryperl.com/download/5.32.1.1/strawberry-perl-no64-5.32.1.1-32bit-portable.zip`     |
| SQLite ![Updated](https://img.shields.io/badge/Updated-green)       | 3.51.0                            | `https://www.sqlite.org/2025/sqlite-tools-win-x64-3510100.zip`                                      |
| winflexbison                                                        | 2.5.25                            | `https://github.com/lexxmark/winflexbison/releases/download/v2.5.25/win_flex_bison-2.5.25.zip`      |

### Building stcopenmpp

In Docker Desktop's terminal (Windows PowerShell), this set of commands creates and starts a container from the previously built Windows image:

<pre>
docker run `
  --rm `
  -it `
  -e OMPP_BUILD_TAG=main `
  -v c:\users\username\downloads\ompp-build:c:\build `
  stcopenmpp/stcopenmpp-build:windows
</pre>

**Note:** `c:\users\username\downloads\ompp-build` represents the pre-existing directory on your local machine where the `stcopenmpp` build will be made available.

The image's default `ENTRYPOINT` is Windows CMD. 

Once the container is running, this command initiates an `stcopenmpp` build:

<pre>
build-all.bat
</pre>

Several environment variables can be set with an `-e` flag (e.g., `-e OMPP_GIT_URL=https://github.com/statcan/stcopenmpp`) when running the image:

| Name                                                                                                       | Description                                   |
| :--------------------------------------------------------------------------------------------------------- | :-------------------------------------------- |
| `OMPP_GIT_URL` Optional (default: `https://github.com/statcan/stcopenmpp`)                                 | The URL to the `stcopenmpp` repo.             |
| `OM_BUILD_CONFIGS` Optional (default: `Release,Debug`)                                                     | The build(s) to make.                         |
| `OM_BUILD_PLATFORMS` Optional (default: `Win32,x64`)                                                       | The build architecture.                       |
| `OM_MSG_USE` Optional (default: empty)                                                                     | If `MPI`, then the MPI version will be built. |
| `OMPP_BUILD_TAG` Optional (default: last commit on default branch)                                         | The repo branch or commit to build from.      |
| `OMPP_DEPLOY_NAME` Optional (default: `stcopenmpp-windows-YYYYMMDD.zip`)                                   | The filename of the build zip.                |
