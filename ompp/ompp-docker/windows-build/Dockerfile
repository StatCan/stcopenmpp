# escape=`

# Docker image to build stcopenmpp latest version on Windows 20H2
# Usage:
#   docker build `
#     --build-arg OMPP_GIT_URL=https://github.com/statcan/stcopenmpp `
#     -t stcopenmpp-build:windows `
#     .

# Declare base image
FROM mcr.microsoft.com/windows/servercore:20H2

# Ensure C:\Temp exists
RUN mkdir C:\Temp

# Download Microsoft Visual Studio Build Tools
RUN curl -L "https://aka.ms/vs/17/release.ltsc.17.4/vs_buildtools.exe" `
  --output C:\Temp\vs_buildtools.exe

# Install Microsoft Visual Studio Build Tools
RUN C:\Temp\vs_buildtools.exe --quiet --wait --norestart `
  --installPath C:\BuildTools `
  --add Microsoft.VisualStudio.Workload.MSBuildTools `
  --add Microsoft.VisualStudio.Workload.VCTools `
  --includeRecommended `
  || IF "%ERRORLEVEL%"=="3010" EXIT 0
  
# Delete installer
RUN del C:\Temp\vs_buildtools.exe

# Download Microsoft MPI
RUN curl -L "https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisetup.exe" `
  --output C:\Temp\msmpisetup.exe

# Install Microsoft MPI
RUN C:\Temp\msmpisetup.exe && del C:\Temp\msmpisetup.exe

# Download Microsoft MPI SDK
RUN curl -L "https://download.microsoft.com/download/7/2/7/72731ebb-b63c-4170-ade7-836966263a8f/msmpisdk.msi" `
  --output C:\Temp\msmpisdk.msi
  
# Install Microsoft MPI SDK
RUN msiexec /i C:\Temp\msmpisdk.msi /q /norestart /L* C:\Temp\msmpisdk.log && `
  del C:\Temp\msmpisdk.msi && `
  del C:\Temp\msmpisdk.log

# Download Git
RUN curl -L "https://github.com/git-for-windows/git/releases/download/v2.52.0.windows.1/Git-2.52.0-64-bit.exe" `
  --output C:\Temp\git_setup.exe

# Set BISON_FLEX_DIR for openM++ build
ENV BISON_FLEX_DIR=C:\bison_flex

# Download WinFlexBison
RUN curl -L "https://github.com/lexxmark/winflexbison/releases/download/v2.5.25/win_flex_bison-2.5.25.zip" `
  --output C:\Temp\winflexbison.zip

# Install WinFlexBison
RUN powershell -NoProfile -Command `
  "Expand-Archive -Path 'C:\Temp\winflexbison.zip' -DestinationPath 'C:\bison_flex'; Remove-Item 'C:\Temp\winflexbison.zip' -Force"
    
# Set SQLITE_EXE_DIR for openM++ build
ENV SQLITE_EXE_DIR=C:\sqlite

# Download SQLite command line tools for Windows
RUN curl -L "https://www.sqlite.org/2025/sqlite-tools-win-x64-3510100.zip" `
  --output C:\Temp\sqlite_bin.zip

# Install SQLite command line tools for Windows
RUN powershell -NoProfile -Command `
  "New-Item -ItemType Directory -Force -Path 'C:\sqlite' | Out-Null; Expand-Archive -Path 'C:\Temp\sqlite_bin.zip' -DestinationPath 'C:\sqlite_tmp'; Get-ChildItem 'C:\sqlite_tmp' -File | Move-Item -Destination 'C:\sqlite' -Force; Remove-Item 'C:\sqlite_tmp' -Recurse -Force; Remove-Item 'C:\Temp\sqlite_bin.zip' -Force"

# Download Go
RUN curl -L "https://go.dev/dl/go1.25.5.windows-amd64.zip" `
  --output C:\Temp\go_setup.zip
  
# Install Go
RUN powershell -NoProfile -Command `
  "Expand-Archive -Path 'C:\Temp\go_setup.zip' -DestinationPath 'C:\'; Remove-Item 'C:\Temp\go_setup.zip' -Force"

# Download MinGW
RUN curl -L "https://nuwen.net/files/mingw/history/mingw-20.0.exe" `
  --output C:\Temp\mingw_setup.exe
  
# Install MinGW
RUN powershell -NoProfile -Command `
  "Expand-Archive -Path 'C:\Temp\mingw_setup.exe' -DestinationPath 'C:\'; Remove-Item 'C:\Temp\mingw_setup.exe' -Force"

# Download Node.js
RUN curl -L "https://nodejs.org/dist/v24.14.0/node-v24.14.0-win-x64.zip" `
  --output C:\Temp\node.zip

# Install Node.js
RUN powershell -NoProfile -Command `
  "Expand-Archive -Path 'C:\Temp\node.zip' -DestinationPath 'C:\'; Rename-Item 'C:\node-v24.14.0-win-x64' 'node'; Remove-Item 'C:\Temp\node.zip' -Force"

# Download Perl portal release (x86)
# Note: 5.32.1.1 is the last x86 release, after that only x64 releases available
RUN curl -L "https://strawberryperl.com/download/5.32.1.1/strawberry-perl-no64-5.32.1.1-32bit-portable.zip" `
  --output C:\Temp\perl32_setup.zip
  
# Install Perl portable release (x86)
RUN powershell -NoProfile -Command `
  "Expand-Archive -Path 'C:\Temp\perl32_setup.zip' -DestinationPath 'C:\perl32'; Remove-Item 'C:\Temp\perl32_setup.zip' -Force"

# Download Perl portal release (x64)
RUN curl -L "https://strawberryperl.com/download/5.32.1.1/strawberry-perl-5.32.1.1-64bit-portable.zip" `
  --output C:\Temp\perl64_setup.zip
  
# Install Perl portable release (x64)
RUN powershell -NoProfile -Command `
  "Expand-Archive -Path 'C:\Temp\perl64_setup.zip' -DestinationPath 'C:\perl64'; Remove-Item 'C:\Temp\perl64_setup.zip' -Force"

# Install CPAN modules (x86)
RUN C:\perl32\portableshell /SETENV && `
  SET TEMP=C:\Temp&&SET TMP=C:\Temp&&cpanm Getopt::Long::Descriptive && cpanm Config::Tiny && cpanm pp

# Install CPAN modules (x64)
RUN C:\perl64\portableshell /SETENV && `
  SET TEMP=C:\Temp&&SET TMP=C:\Temp&&cpanm Getopt::Long::Descriptive && cpanm Config::Tiny && cpanm pp

# Set local timezone
RUN tzutil /s "Eastern Standard Time"

# Copy build scripts
COPY ["copy_scripts.bat","build-all.bat","build-openm.bat","build-models.bat","build-go.bat","build-ui.bat","build-perl.bat","build-zip.bat","make-model.bat","README.md","C:\\build_scripts\\"]

# Run subsequent commands as the built-in administrator account
# This is required to set machine-wide environment variables and modify PATH
USER ContainerAdministrator

# Configure machine-level environment variables for the build toolchain
RUN setx /M GOROOT C:\go && `
  setx /M GOPATH C:\build\ompp && `
  setx /M OM_SCRIPTS_DIR C:\build_scripts && `
  setx /M PATH "%GOROOT%\bin;%GOPATH%\bin;C:\Git\cmd;C:\R\bin;%PATH%"

# Drop privileges back to the default non-admin user
# This follows least-privilege best practices and prevents accidental
USER ContainerUser

# Set image labels
LABEL name=stcopenmpp-build:windows
LABEL os=Windows
LABEL license=MIT
LABEL description="stcopenmpp build environemnt: VC++ 2022, MSBuild, Microsoft MPI, Git, SQLite, bison, flex, Go, MinGW, node.js, Perl"

# Set image build variables 
ARG OMPP_GIT_URL=https://github.com/statcan/stcopenmpp

# Set the default working directory inside the container
# All relative paths will be resolved from here
WORKDIR /build

# Entry point initializes all required build environments:
ENTRYPOINT ["cmd.exe", "/S", "/C", "C:\\BuildTools\\Common7\\Tools\\VsDevCmd.bat && C:\\MinGW\\set_distro_paths.bat && C:\\node\\nodevars.bat && %OM_SCRIPTS_DIR%\\copy_scripts.bat &&"]

# Default CMD opens cmd.exe
CMD ["cmd.exe"]