@echo off
REM build openM++ run-time libraries and omc compiler
REM
REM environmemnt variables:
REM  set OM_BUILD_CONFIGS=Release,Debug (default: Release,Debug)
REM  set OM_BUILD_PLATFORMS=Win32,x64   (default: Win32,x64)
REM  set OM_MSG_USE=MPI                 (default: EMPTY)
REM  set OMPP_CPP_BUILD_TAG             (default: build from latest git)
REM  set OMPP_BUILD_TAG                 (default: build from latest git)
REM  set OMPP_GIT_URL                   (default: https://github.com/statcan/stcopenmpp)
REM  set OMPP_DEPLOY_NAME               (default: stcopenmpp-windows-YYYYMMDD)

setlocal enabledelayedexpansion

set OM_BLD_CFG=Release,Debug
set OM_BLD_PLT=Win32,x64

if defined OM_BUILD_CONFIGS   set OM_BLD_CFG=%OM_BUILD_CONFIGS%
if defined OM_BUILD_PLATFORMS set OM_BLD_PLT=%OM_BUILD_PLATFORMS%
if /I "%OM_MSG_USE%"=="MPI"   set OM_P_MPI=-p:OM_MSG_USE=MPI

REM show environment

set START_DT=%DATE% %TIME%
@echo %START_DT% Build openM++ run-time libraries and omc compiler
@echo Environment:
@echo OM_BUILD_CONFIGS   = %OM_BUILD_CONFIGS%
@echo OM_BUILD_PLATFORMS = %OM_BUILD_PLATFORMS%
@echo OM_MSG_USE         = %OM_MSG_USE%
@echo Build configurations: %OM_BLD_CFG%
@echo Build platforms:      %OM_BLD_PLT%
if defined OM_P_MPI (
  @echo Build cluster version: using MPI
) else (
  @echo Build desktop version: non-MPI
)

REM get source code from git
if not exist stcopenmpp/ompp (
  REM clone repo
  @echo git clone %OMPP_GIT_URL% stcopenmpp
  git clone %OMPP_GIT_URL% stcopenmpp
  
  REM exit on error
  if ERRORLEVEL 1 (
    @echo FAILED.
    EXIT
  ) 
  
  REM change directory
  pushd C:\build\stcopenmpp
  
  REM fix issue with dubious ownership (e.g., fatal: detected dubious ownership in repository at 'C:/build/stcopenmpp')
  git config --global --add safe.directory *
  
  REM make log directory
  if not exist ompp\log mkdir ompp\log

  REM if OMPP_CPP_BUILD_TAG or OMPP_BUILD_TAG is set then build from that git tag or branch
  if defined OMPP_BUILD_TAG (
    set OM_BLD_TAG=%OMPP_BUILD_TAG%
    @echo OMPP_BUILD_TAG     = %OMPP_BUILD_TAG%
    @echo OMPP_BUILD_TAG     = %OMPP_BUILD_TAG% >> ompp\log\build-openm.log
  )
  
  if defined OMPP_CPP_BUILD_TAG (
    set OM_BLD_TAG=%OMPP_CPP_BUILD_TAG%
    @echo OMPP_CPP_BUILD_TAG = %OMPP_CPP_BUILD_TAG%
    @echo OMPP_CPP_BUILD_TAG = %OMPP_CPP_BUILD_TAG% >> ompp\log\build-openm.log
  )

  if defined OM_BLD_TAG (
    @echo git checkout !OM_BLD_TAG!
    @echo git checkout !OM_BLD_TAG! >> ompp\log\build-openm.log
    git checkout !OM_BLD_TAG! >> ompp\log\build-openm.log 2>&1
	
	REM exit on error
    if ERRORLEVEL 1 (
      @echo FAILED: git checkout !OM_BLD_TAG! >> ompp\log\build-openm.log
      @echo FAILED.
      EXIT
    )
  )

  REM find openM++ version commit and use commit tag, if tagged
  @echo git log -n 1 --date=short --format.... >> ompp\log\build-openm.log
  for /F "usebackq tokens=* delims=" %%i in (`git log -n 1 --date^=short --format^="%%cd %%H"`) do (
    REM exit on error
	if ERRORLEVEL 1 (
      @echo FAILED.
      EXIT
    )
    set OM_RUNTIME_VERSION=%%i
  )

  if defined OM_BLD_TAG (
    for /F "usebackq tokens=* delims=" %%i in (`git tag -l %OM_BLD_TAG%`) do (
      if ERRORLEVEL 1 (
        @echo FAILED: git tag -l %OM_BLD_TAG%
        @echo FAILED: git tag -l %OM_BLD_TAG% >> ompp\log\build-openm.log
        @echo FAILED.
        EXIT
      )
      if /i "%%i"=="%OM_BLD_TAG%" (
        set OM_RUNTIME_VERSION=%OM_RUNTIME_VERSION% %OM_BLD_TAG%
      )
    )
  )

  @echo OM_RUNTIME_VERSION = !OM_RUNTIME_VERSION! >> ompp\log\build-openm.log
  @echo OM_RUNTIME_VERSION = !OM_RUNTIME_VERSION!
  
  REM move all contents from %OMPP_GIT_URL%/ompp to ompp
  robocopy "C:\build\stcopenmpp\ompp" "C:\build\ompp" /E /MOVE
  
  REM move all contents from %OMPP_GIT_URL%/plugins to ompp/plugins
  robocopy "C:\build\stcopenmpp\plugins" "C:\build\ompp\plugins" /E /MOVE
  
  REM exit on error
  if %ERRORLEVEL% GEQ 8 (
	@echo FAILED moving ompp contents from %OMPP_GIT_URL%.
	EXIT
  )
) else (
  @echo Skip: git clone
)

if defined OMPP_DEPLOY_NAME (
  set OMPP_DEPLOY_NAME=%OMPP_DEPLOY_NAME%
  @echo OMPP_DEPLOY_NAME     = %OMPP_DEPLOY_NAME%
  @echo OMPP_DEPLOY_NAME     = %OMPP_DEPLOY_NAME% >> ompp\log\build-openm.log
)

REM push into ompp root
pushd C:\build\ompp
set OM_ROOT=%CD%
@echo OM_ROOT = %OM_ROOT%

REM remove stcopenmpp
rmdir /s /q C:\build\stcopenmpp

REM log build environment 

@echo Log file: log\build-openm.log
@echo %START_DT% Build openM++ run-time libraries and omc compiler >> log\build-openm.log
@echo OM_BUILD_CONFIGS   = %OM_BUILD_CONFIGS% >> log\build-openm.log
@echo OM_BUILD_PLATFORMS = %OM_BUILD_PLATFORMS% >> log\build-openm.log
@echo OM_MSG_USE         = %OM_MSG_USE% >> log\build-openm.log
@echo OM_ROOT            = %OM_ROOT% >> log\build-openm.log
@echo Build configurations: %OM_BLD_CFG% >> log\build-openm.log
@echo Build platforms:      %OM_BLD_PLT% >> log\build-openm.log
if defined OM_P_MPI (
  @echo Build cluster version: using MPI >> log\build-openm.log
) else (
  @echo Build desktop version: non-MPI >> log\build-openm.log
)

REM create omVersion.h

@echo Create include/libopenm/omVersion.h >> log\build-openm.log

@echo /** > include/libopenm/omVersion.h
@echo  * @file >> include/libopenm/omVersion.h
@echo  * OpenM++: runtime version >> include/libopenm/omVersion.h
@echo  */ >> include/libopenm/omVersion.h
@echo // Copyright (c) OpenM++ >> include/libopenm/omVersion.h
@echo // This code is licensed under the MIT license (see LICENSE.txt for details) >> include/libopenm/omVersion.h
@echo // >> include/libopenm/omVersion.h
@echo #ifndef OM_H_VERSION_H >> include/libopenm/omVersion.h
@echo #define OM_H_VERSION_H >> include/libopenm/omVersion.h
@echo // >> include/libopenm/omVersion.h

@echo #define OM_RUNTIME_VERSION "%OM_RUNTIME_VERSION%" >> include/libopenm/omVersion.h
 
@echo // >> include/libopenm/omVersion.h
@echo #endif  // OM_H_VERSION_H >> include/libopenm/omVersion.h

REM build c++ run-time libraries and omc compiler

pushd openm
for %%c in (%OM_BLD_CFG%) do (
  for %%p in (%OM_BLD_PLT%) do (

    REM build model runtime libraries

    call :make_openm_sln "%OM_P_MPI% -p:Configuration=%%c -p:Platform=%%p openm.sln /target:libsqlite:Rebuild"
    call :make_openm_sln "%OM_P_MPI% -p:Configuration=%%c -p:Platform=%%p openm.sln /target:libopenm:Rebuild"

    REM build omc model compiler

    if /i "%%c"=="Release" (
      if /i "%%p"=="x64" (
        call :make_openm_sln "-p:Configuration=%%c -p:Platform=%%p openm.sln /target:libopenm_omc_db:Rebuild"
        call :make_openm_sln "-p:Configuration=%%c -p:Platform=%%p openm.sln /target:omc:Rebuild"
      )
    )

    REM build libopenmD_disable_iterator_debug: non-default iterator debug level

    if /i "%%c"=="Debug" (

      if exist ..\build\libopenm (

        @echo Remove ..\build\libopenm >> ..\log\build-openm.log

        for /L %%k in (1,1,8) do (
          if exist ..\build\libopenm (
            rd /s /q ..\build\libopenm >> ..\log\build-openm.log 2>&1
          )
          if exist ..\build\libopenm (
            ping 127.0.0.1 -n 2 -w 500 >nul
          )
        )
        if exist ..\build\libopenm (
          @echo FAILED to delete: ..\build\libopenm
          @echo FAILED to delete: ..\build\libopenm >> ..\log\build-openm.log
          EXIT 1
        )
      )

      call :make_openm_sln "%OM_P_MPI% -p:Configuration=%%c -p:Platform=%%p -p:DISABLE_ITERATOR_DEBUG=true openm.sln /target:libopenm"
    )
  )
)
popd

@echo %DATE% %TIME% Done.
@echo %DATE% %TIME% Done. >> log\build-openm.log

popd
goto :eof

REM end of main body

REM build openm solution subroutine
REM arguments: 
REM  1 = msbuild command line arguments

:make_openm_sln

set mk_args=%~1
@echo msbuild %mk_args%
@echo msbuild %mk_args% >> ..\log\build-openm.log

msbuild %mk_args% >> ..\log\build-openm.log 2>&1
if ERRORLEVEL 1 (
  @echo FAILED.
  @echo FAILED. >> ..\log\build-openm.log
  EXIT
) 
exit /b
