@echo off
REM copy UI and binary utilities (e.g. oms.exe, dbcopy.exe) from current location to %OM_ROOT% directory

setlocal enabledelayedexpansion

if "%CD%\" == "%~dp0" (
  pushd ..
)

REM check destination directory, expected to be at least git clone of GitHub openmpp/main

if not defined OM_ROOT (
  @echo ERROR: destination directory not defined, do: SET OM_ROOT=\some\dir
  pause
  EXIT 1
)
if not exist "%OM_ROOT%" (
  @echo ERROR: destination directory not exist: %OM_ROOT%
  @echo        may be you want to do:
  @echo .
  @echo   git clone %OMPP_GIT_URL%/%OMPP_MAIN_GIT% %OM_ROOT%
  @echo .
  pause
  EXIT 1
)

REM create destination directories, if not exist

if not exist "%OM_ROOT%\etc"                     call :make_dir "%OM_ROOT%\etc"
if not exist "%OM_ROOT%\log"                     call :make_dir "%OM_ROOT%\log"
if not exist "%OM_ROOT%\models\bin"              call :make_dir "%OM_ROOT%\models\bin"
if not exist "%OM_ROOT%\models\log"              call :make_dir "%OM_ROOT%\models\log"
if not exist "%OM_ROOT%\models\doc"              call :make_dir "%OM_ROOT%\models\doc"
if not exist "%OM_ROOT%\models\home\io\download" call :make_dir "%OM_ROOT%\models\home\io\download"
if not exist "%OM_ROOT%\models\home\io\upload"   call :make_dir "%OM_ROOT%\models\home\io\upload"

REM copy bin utilities

call :rcopy_files "%OM_ROOT%\bin" bin "dbcopy.* oms.* dbget.* launch-ui-windows.bat ompp_ui.bat sqlite3.exe"

REM copy etc templates

call :rcopy_files "%OM_ROOT%\etc" etc "*.*"

REM delete existing html directory

if exist "%OM_ROOT%\html" (

  @echo Remove existing: %OM_ROOT%\html
  
  for /L %%k in (1,1,8) do (
    if exist "%OM_ROOT%\html" (
      rd /s /q "%OM_ROOT%\html"
    )
    if exist "%OM_ROOT%\html" (
      ping 127.0.0.1 -n 2 -w 500 >nul
    )
  )

  if exist "%OM_ROOT%\html" (
    @echo FAILED to delete directory: "%OM_ROOT%\html"
    pause
    EXIT 1
  )
)

REM copy new html directory

call :rcopy_sub_dirs "%OM_ROOT%" . html

REM

popd
goto :eof

REM end of main body

REM robocopy file(s)
REM arguments:
REM  1 = destination directory
REM  2 = source directory
REM  3 = file name(s), space separated, must be "quoted", can be wildcard

:rcopy_files

set dst_dir=%1
set src_dir=%2
set fnames=%~3

@echo Copy files from: %src_dir%

robocopy /njh /njs /np %src_dir% %dst_dir% %fnames%
if ERRORLEVEL 2 (
  @echo FAIL to copy: %fnames%
  pause
  EXIT
)
exit /b

REM robocopy sub-directories
REM arguments:
REM  1 = destination directory
REM  2 = source root directory
REM  3 = source sub-directories, comma or space separated list, must be "quoted"

:rcopy_sub_dirs

set dst_dir=%1
set src_root=%2
set src_dir_lst=%~3

for %%d in (%src_dir_lst%) do (

  @echo Copy directory: %%d
  
  robocopy /s /e /njh /njs /np %src_root%\%%d %dst_dir%\%%d
  if ERRORLEVEL 2 (
    @echo FAIL to copy: %%d
    pause
    EXIT
  )
)
exit /b

REM create directory, exit on error, do log
REM arguments:
REM  1 = destination directory

:make_dir

set dst_dir=%1

@echo Create directory: %dst_dir%

mkdir %dst_dir%
if ERRORLEVEL 1 (
  @echo FAIL to create: %dst_dir%
  pause
  EXIT
)
exit /b
