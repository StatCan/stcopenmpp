@echo off

REM Copy model files from source $src_root to destination $dst_root, for example:
REM
REM model-copy.bat RiskPaths.publish.lst D:\archive C:\my-work RiskPaths v3.2.1
REM
REM Arguments:
REM
REM   %1 - publish_lst: path to publish list file, if relative then must be relative to source $src_root
REM   %2 - src_root   : source root path, if relative then must be relative to OM_ROOT
REM   %3 - dst_root   : destination root, if relative then must be relative to OM_ROOT
REM   %4 - name_ver   : model name or name-version e.g.: RiskPaths-v3.2.1
REM
REM It does:
REM   - reads list of model files from $publish_lst, e.g.: from RiskPaths.publish.lst
REM   - create destination directories (if not exists) and copy files
REM   - save list of files into model copy list, e.g.: RiskPaths-v3.2.1.copy.lst
REM
REM Environment:
REM
REM   OM_ROOT                       (optional) openM++ root path
REM   BIN_DIR  default: models\bin  sub-folder where model.exe and model.sqlite resides
REM   DOC_DIR  default: models\doc  models documentation sub-folder
REM   LOG_DIR  default: models\log  models log sub-folder
REM
REM Example:
REM
REM model-copy.bat models\bin\RiskPaths.publish.lst D:\archive C:\my-work RiskPaths
REM
REM where D:\archive\models\bin\RiskPaths.publish.lst :
REM
REM   $BIN_DIR\RiskPaths.exe
REM   $BIN_DIR\RiskPaths.sqlite
REM   $DOC_DIR\RiskPaths.doc.EN.html
REM   $DOC_DIR\RiskPaths.doc.FR.html
REM   $LOG_DIR\RiskPaths.log
REM   some_data.file
REM
REM It does copy from => to:
REM
REM D:\archive\models\bin\RiskPaths.exe         => C:\my-work\models\bin\RiskPaths.exe
REM D:\archive\models\bin\RiskPaths.sqlite      => C:\my-work\models\bin\RiskPaths.sqlite
REM D:\archive\models\doc\RiskPaths.doc.EN.html => C:\my-work\models\doc\RiskPaths.doc.EN.html
REM D:\archive\models\doc\RiskPaths.doc.FR.html => C:\my-work\models\doc\RiskPaths.doc.FR.html
REM D:\archive\models\log\RiskPaths.log         => C:\my-work\models\log\RiskPaths.log
REM D:\archive\some_data.file                   => C:\my-work\some_data.file

setlocal enabledelayedexpansion

set "publish_lst=%1"
set "src_root=%2"
set "dst_root=%3"
set "name_ver=%4"

IF not defined src_root (
  @echo ERROR: invalid or empty source directory: "%src_root%"
  EXIT 1
)
IF not defined dst_root (
  @echo ERROR: invalid or empty destination directory: "%dst_root%"
  EXIT 1
)
IF not defined publish_lst (
  @echo ERROR: invalid or empty model files list: "%publish_lst%"
  EXIT 1
)
IF not defined name_ver (
  @echo ERROR: invalid or empty model name-version: "%name_ver%"
  EXIT 1
)

IF "%src_root:~-1%"=="\" set "src_root=%src_root:~0,-1%"
IF "%dst_root:~-1%"=="\" set "dst_root=%dst_root:~0,-1%"

IF "%src_root%" == "" (
  @echo ERROR: invalid or empty source directory: "%src_root%"
  EXIT 1
)
IF "%dst_root%" == "" (
  @echo ERROR: invalid or empty destination directory: "%dst_root%"
  EXIT 1
)
IF "%publish_lst%" == "" (
  @echo ERROR: invalid or empty model files list: "%publish_lst%"
  EXIT 1
)

REM set model files sub-directories, if not defined then use defaults

IF not defined BIN_DIR set BIN_DIR=models\bin
IF not defined DOC_DIR set DOC_DIR=models\doc
IF not defined LOG_DIR set LOG_DIR=models\log

IF "%BIN_DIR:~-1%"=="\" set "BIN_DIR=%BIN_DIR:~0,-1%"
IF "%DOC_DIR:~-1%"=="\" set "DOC_DIR=%DOC_DIR:~0,-1%"
IF "%LOG_DIR:~-1%"=="\" set "LOG_DIR=%LOG_DIR:~0,-1%"

@echo Model   : %name_ver%
@echo Copy    : %publish_lst%
@echo From    : %src_root%
@echo To      : %dst_root%
@echo OM_ROOT : %OM_ROOT%
@echo BIN_DIR : %BIN_DIR%
@echo DOC_DIR : %DOC_DIR%
@echo LOG_DIR : %LOG_DIR%

REM if defined OM_ROOT then pushd to OM_ROOT

IF defined OM_ROOT (
  @echo pushd %OM_ROOT%

  pushd "%OM_ROOT%"
  if ERRORLEVEL 1 (
    @echo FAILED: pushd "%OM_ROOT%"
    @echo FAILED.
    EXIT 1
  )
)

FOR %%A in ("%src_root%") DO set "abs_src_root=%%~fA"
FOR %%A in ("%dst_root%") DO set "abs_dst_root=%%~fA"

IF "%abs_src_root:~-1%"=="\" set "abs_src_root=%abs_src_root:~0,-1%"
IF "%abs_dst_root:~-1%"=="\" set "abs_src_root=%abs_dst_root:~0,-1%"

IF /i "%abs_src_root%" == "%abs_dst_root%" (
  @echo ERROR: source and destination directory are the same: "%abs_src_root%"
  EXIT 1
)

REM check if source root exist

IF not exist "%src_root%" (
  @echo ERROR: source directory not found or invalid: "%src_root%"
  EXIT 1
)
IF not exist "%abs_src_root%\" (
  @echo ERROR: source directory not found or invalid: "%abs_src_root%"
  EXIT 1
)

REM create destination root if not exist

IF not exist "%dst_root%" (

  mkdir "%dst_root%"
  IF ERRORLEVEL 1 (
    @echo FAILED: mkdir "%dst_root%"
    @echo FAILED.
    EXIT 1
  )
)
IF not exist "%abs_dst_root%\" (
  @echo "ERROR: destination directory not found or invalid: %abs_dst_root%"
  EXIT 1
)

REM pushd to destination root

@echo pushd %dst_root%

pushd "%dst_root%"
if ERRORLEVEL 1 (
  @echo FAILED: pushd "%dst_root%"
  @echo FAILED.
  EXIT 1
)


REM if path to ModelName.publish.lst is relative then make an absolute path

FOR %%A in ("%publish_lst%") DO set "t_p_l=%%~fA"

if /i "%publish_lst%" == "%t_p_l%" (
  set "abs_pub_lst=%t_p_l%"
) else (
  set "abs_pub_lst=%abs_src_root%\%publish_lst%"
)

IF not exist "%abs_pub_lst%" (
  @echo ERROR: model files list file not found or invalid: "%abs_pub_lst%"
  EXIT 1
)

REM save copy of file list into dst_root\BIN_DIR\ModelName-version.copy.lst

set "abs_cp_lst=%abs_dst_root%\%BIN_DIR%\%name_ver%.copy.lst"

IF exist "%abs_cp_lst%" (

  del /F /Q "%abs_cp_lst%"

  IF ERRORLEVEL 1 (
    REM del do NOT retrun error if delete failed. 
    @echo FAILED to delete "%abs_cp_lst%"
    @echo FAILED.
    EXIT 1
  )
)

REM copy files and save file list in ModelName.publish.lst

set wd=%CD%

FOR /F "usebackq tokens=* delims=" %%f in ("%abs_pub_lst%") DO (

  REM replace $BIN_DIR $DOC_DIR $LOG_DIR with actual path

  set "ln=%%f"
  set "ln=!ln:/=\!"
  set "ln=!ln:$BIN_DIR=%BIN_DIR%!"
  set "ln=!ln:$DOC_DIR=%DOC_DIR%!"
  set "ln=!ln:$LOG_DIR=%LOG_DIR%!"

  REM get destination directory absolute path and file name
  
  FOR %%A in ("!ln!") DO (
    set "a_t_dir=%%~dpA"
    set "fn=%%~nxA"
  )

  REM make source and destination directory absolute path

  set "r_dir=!a_t_dir:%wd%=!"

  IF "!r_dir:~0,1!" == "\" (
    set "a_f_dir=%abs_src_root%!r_dir!"
  ) else (
    set "a_f_dir=%abs_src_root%\!r_dir!"
  )

  IF "!a_f_dir:~-1!"=="\" set "a_f_dir=!a_f_dir:~0,-1!"
  IF "!a_t_dir:~-1!"=="\" set "a_t_dir=!a_t_dir:~0,-1!"

  REM check: source file must be under source root

  IF not exist "!a_f_dir!\!fn!" (
    @echo ERROR: source file or directory not found: "!a_f_dir!\!fn!"
    EXIT 1
  )
  IF not exist "%abs_src_root%\!r_dir!\!fn!" (
    @echo ERROR: source file or directory not found: "%abs_src_root%\!r_dir!\!fn!"
    EXIT 1
  )

  REM create destination directory and copy file

  IF not exist "!a_t_dir!" (
    mkdir "!a_t_dir!"
    IF ERRORLEVEL 1 (
      @echo FAILED: mkdir "!a_t_dir!"
      @echo FAILED.
      EXIT 1
    )
  )

  REM check if source is directory or file
  
  IF exist "!a_f_dir!\!fn!\" (
    set "cm=robocopy /s /e /is /im /is /xx /j /r:0 /w:0 /njh /njs /np !a_f_dir!\!fn! !a_t_dir!\!fn!"
    @echo !cm!

    !cm!
    if ERRORLEVEL 8 (
      @echo FAILED: !cm!
      @echo FAILED.
      EXIT 1
    )
  ) ELSE (
    set "cm=copy /b !a_f_dir!\!fn! !a_t_dir!\!fn!"
    @echo !cm!

    !cm!
    if ERRORLEVEL 1 (
      @echo FAILED: !cm!
      @echo FAILED.
      EXIT 1
    )
  )
  @echo !a_t_dir!\!fn! >> "%abs_cp_lst%"
)

@echo Done.

EXIT /b 0
