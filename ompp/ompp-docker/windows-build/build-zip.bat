@echo off
REM create zip archive of stcopenmpp build from ompp sub-directory: stcopenmpp-windows-20180817.zip
REM environmemnt variables:
REM
REM  set OM_MSG_USE=MPI                 (default: EMPTY)
REM  set OM_DATE_STAMP=20220817         (default: current date as YYYYMMDD)
REM  set MODEL_DIRS=modelOne,NewCaseBased,NewTimeBased,NewCaseBased_bilingual,IDMM,RiskPaths,OzProjX,OzProjGenX,SM1

setlocal enabledelayedexpansion

if /I "%OM_MSG_USE%"=="MPI" set OM_SFX_MPI=_mpi

set OM_BLD_MDLS=modelOne,NewCaseBased,NewTimeBased,NewCaseBased_bilingual,IDMM,RiskPaths,OzProjX,OzProjGenX,SM1
if defined MODEL_DIRS       set OM_BLD_MDLS=%MODEL_DIRS%

REM push into ompp root and make log directory if not exist

if not exist ompp (
  @echo ERROR: missing source code directory: ompp
  EXIT 1
)

pushd ompp
set   OM_ROOT=%CD%

if not exist log mkdir log

REM if OM_DATE_STAMP is not defined then use current local date
for /f "tokens=2 delims==." %%G in ('wmic OS Get localdatetime /value') do (
  set local_ts=%%G
)

if not defined OM_DATE_STAMP set OM_DATE_STAMP=%local_ts:~0,8%

REM Check if DEPLOY_NAME variable exists
IF DEFINED OMPP_DEPLOY_NAME (
    SET DEPLOY_DIR=..\%OMPP_DEPLOY_NAME%
) ELSE (
    SET DEPLOY_DIR=..\stcopenmpp-windows%OM_SFX_MPI%-%OM_DATE_STAMP%
)

set DEPLOY_ZIP=%DEPLOY_DIR%.zip

REM log build environment 

@echo %DATE% %TIME% Pack openM++ build
@echo Environment:
@echo  OM_MSG_USE    = %OM_MSG_USE%
@echo  OM_ROOT       = %OM_ROOT%
@echo  MODEL_DIRS    = %MODEL_DIRS%
@echo  OM_DATE_STAMP = %OM_DATE_STAMP%
@echo  DEPLOY_DIR    = %DEPLOY_DIR%
@echo Pack into: %DEPLOY_ZIP%

@echo Log file: log\build-zip.log
@echo %DATE% %TIME% Pack openM++ build > log\build-zip.log
@echo  OM_MSG_USE    = %OM_MSG_USE% >> log\build-zip.log
@echo  OM_ROOT       = %OM_ROOT% >> log\build-zip.log
@echo  MODEL_DIRS    = %MODEL_DIRS% >> log\build-zip.log
@echo  OM_DATE_STAMP = %OM_DATE_STAMP% >> log\build-zip.log
@echo  DEPLOY_DIR    = %DEPLOY_DIR% >> log\build-zip.log
@echo Pack into: %DEPLOY_ZIP% >> log\build-zip.log

REM fix git clone issue:
REM ....fatal: detected dubious ownership in repository at 'C:/build/ompp'

@echo git config --global --add safe.directory *

git config --global --add safe.directory * >> log\build-zip.log 2>&1
if ERRORLEVEL 1 (
  @echo FAILED.
  EXIT
) 

REM delete existing pack directory and zip file

if exist %DEPLOY_DIR% (

  @echo Remove existing: %DEPLOY_DIR%
  @echo Remove existing: %DEPLOY_DIR% >> log\build-zip.log
  
  for /L %%k in (1,1,8) do (
    if exist %DEPLOY_DIR% (
      rd /s /q %DEPLOY_DIR% >> log\build-zip.log 2>&1
    )
    if exist %DEPLOY_DIR% (
      ping 127.0.0.1 -n 2 -w 500 >nul
    )
  )

  if exist %DEPLOY_DIR% (
    @echo FAILED to delete: %DEPLOY_DIR%
    @echo FAILED to delete: %DEPLOY_DIR% >> log\build-zip.log
    EXIT 1
  )
)

if exist %DEPLOY_ZIP% (

  @echo Remove existing: %DEPLOY_ZIP%
  @echo Remove existing: %DEPLOY_ZIP% >> log\build-zip.log
  
  for /L %%k in (1,1,8) do (
    if exist %DEPLOY_ZIP% (
      del /f /q %DEPLOY_ZIP% >> log\build-zip.log 2>&1
    )
    if exist %DEPLOY_ZIP% (
      ping 127.0.0.1 -n 2 -w 500 >nul
    )
  )

  if exist %DEPLOY_ZIP% (
    @echo FAIL to delete: %DEPLOY_ZIP%
    @echo FAIL to delete: %DEPLOY_ZIP% >> log\build-zip.log
    EXIT 1
  )
)

REM create new deploy directory, copy top level sub-directories and files

call :make_dir %DEPLOY_DIR%

echo Copy files: > log\build-zip-copy.log

call :rcopy_files    %DEPLOY_DIR% . "*.*"
call :rcopy_sub_dirs %DEPLOY_DIR% . "Excel,include,licenses,openm,Perl,plugins,props,sql,use,Xcode"

REM copy openm runtime libraries

set DID_SFX=_disable_iterator_debug

call :rcopy_files ^
  %DEPLOY_DIR%\lib ^
  lib ^
  "libsqlite.lib libsqlite32.lib libsqliteD.lib libsqlite32D.lib"

call :rcopy_files ^
  %DEPLOY_DIR%\lib ^
  lib ^
  "libopenm%OM_SFX_MPI%.lib libopenm32%OM_SFX_MPI%.lib libopenmD%OM_SFX_MPI%.lib libopenm32D%OM_SFX_MPI%.lib libopenmD%OM_SFX_MPI%%DID_SFX%.lib libopenm32D%OM_SFX_MPI%%DID_SFX%.lib"

REM copy bin directory executables

call :rcopy_files %DEPLOY_DIR%\bin bin "omc.exe omc.ini omc.message.ini sqlite3.exe"
call :rcopy_files %DEPLOY_DIR%\bin bin "launch-ui-linux.sh launch-ui-windows.bat ompp_ui.bat ompp_ui_linux.sh ompp_ui_mac.command start_oms.sh README_win.txt"

call :rcopy_files ^
  %DEPLOY_DIR%\bin ^
  bin ^
  "test_models.exe CsvToDat.exe create_import_set.exe ompp_create_scex.exe ompp_export_csv.exe ompp_export_excel.exe modgen_export_csv.exe patch_modgen11_outputs.exe patch_modgen12_outputs.exe patch_modgen12.1_outputs.exe"

call :rcopy_files ^
  %DEPLOY_DIR%\bin ^
  bin ^
  "test_models32.exe CsvToDat32.exe create_import_set32.exe ompp_create_scex32.exe ompp_export_csv32.exe ompp_export_excel32.exe modgen_export_csv32.exe patch_modgen11_outputs32.exe patch_modgen12_outputs32.exe patch_modgen12.1_outputs32.exe"

REM copy Go bin executables and source code

call :rcopy_files    %DEPLOY_DIR%\bin     bin     "dbcopy.exe dbget.exe oms.exe"
call :rcopy_files    %DEPLOY_DIR%\bin     ompp-go "dbcopy.message.ini dbget.message.ini oms.message.ini go-common.message.ini"
call :rcopy_files    %DEPLOY_DIR%\ompp-go ompp-go "*.*"
call :rcopy_sub_dirs %DEPLOY_DIR%\ompp-go ompp-go "dbcopy,dbget,etc,job,licenses,models,ompp,oms"

REM copy template files to run models

call :make_dir %DEPLOY_DIR%\etc

call :do_copy_files %DEPLOY_DIR%\etc\run.Debug.template.txt      ompp-go\etc\run-Windows.Debug.template.txt
call :do_copy_files %DEPLOY_DIR%\etc\mpi.ModelRun.template.txt   ompp-go\etc\mpi-Windows.ModelRun.template.txt
call :do_copy_files %DEPLOY_DIR%\etc\mpi.ModelDebug.template.txt ompp-go\etc\mpi-Windows.ModelDebug.template.txt
call :do_copy_files %DEPLOY_DIR%\etc\disk.ini                    ompp-go\etc\disk_win.ini
call :do_copy_files %DEPLOY_DIR%\etc\                            ompp-go\etc\db-cleanup.bat
call :do_copy_files %DEPLOY_DIR%\etc\                            ompp-go\etc\ui.extra.json
call :do_copy_files %DEPLOY_DIR%\etc\                            ompp-go\etc\run-options.any_model.1.Use_Defaults.json

REM copy UI html build and source code

call :rcopy_sub_dirs %DEPLOY_DIR%    ompp-ui\dist "spa"
call :do_cmd_line_log log\build-zip.log "rename %DEPLOY_DIR%\spa html"
call :rcopy_files    %DEPLOY_DIR%\ompp-ui ompp-ui          "*.*"
call :rcopy_sub_dirs %DEPLOY_DIR%\ompp-ui ompp-ui          ".vscode,licenses,public,src"

REM create log directories and models directories

call :make_dir %DEPLOY_DIR%\log
call :make_dir %DEPLOY_DIR%\models\bin
call :make_dir %DEPLOY_DIR%\models\sql
call :make_dir %DEPLOY_DIR%\models\log
call :make_dir %DEPLOY_DIR%\models\doc
call :make_dir %DEPLOY_DIR%\models\home\io\download
call :make_dir %DEPLOY_DIR%\models\home\io\upload

REM copy models
REM modelOne special case:
REM   it does not have modgen or any code/*.ompp or parameters/*.mpp files

call :do_copy_files  %DEPLOY_DIR%\models models\*.*

for %%m in (%OM_BLD_MDLS%) do (
  
  set MDL_DIR=%%m
  @echo Copy model: %%m
  @echo Copy model: %%m >> log\build-zip.log
  
  if /i "!MDL_DIR:modelOne=!"=="!MDL_DIR!" (
  
    call :rcopy_files    %DEPLOY_DIR%\models\%%m        models\%%m          "*.*"
    call :rcopy_sub_dirs %DEPLOY_DIR%\models\%%m        models\%%m          "code,parameters"
    call :rcopy_files    %DEPLOY_DIR%\models\%%m\modgen models\%%m\modgen   "*.vcxproj *.vcxproj.filters *.props"
    call :do_copy_files  %DEPLOY_DIR%\models\sql        models\%%m\ompp\src\*.sql

    if exist models\%%m\microdata (
      call :rcopy_sub_dirs %DEPLOY_DIR%\models\%%m  models\%%m  "microdata"
    )
    if exist models\%%m\code_original (
      call :rcopy_sub_dirs %DEPLOY_DIR%\models\%%m  models\%%m  "code_original"
    )
    if exist models\%%m\doc (
      call :rcopy_sub_dirs %DEPLOY_DIR%\models\%%m  models\%%m  "doc"
    )
    if exist models\%%m\ompp\bin\doc (
      call :do_copy_files  %DEPLOY_DIR%\models\doc   models\%%m\ompp\bin\doc\*.*
      call :do_copy_files  %DEPLOY_DIR%\models\bin   models\%%m\ompp\bin\%%m.extra.json
    )

  ) else (
  
    call :rcopy_files    %DEPLOY_DIR%\models\%%m models\%%m  "*.*"
    call :rcopy_sub_dirs %DEPLOY_DIR%\models\%%m models\%%m  "csv,tsv,csv_id,tsv_id"
    call :do_copy_files  %DEPLOY_DIR%\models\sql models\%%m\*.sql
  )
  
  call :rcopy_files %DEPLOY_DIR%\models\%%m\ompp models\%%m\ompp "*.vcxproj *.vcxproj.filters *.props"
    
  call :rcopy_files ^
    %DEPLOY_DIR%\models\bin ^
    models\%%m\ompp\bin ^
    "!MDL_DIR!%OM_SFX_MPI%.exe !MDL_DIR!.sqlite !MDL_DIR!*.ini"

  call :rcopy_files %DEPLOY_DIR%\models\log models\%%m\ompp\bin "!MDL_DIR!.*.log"
)

REM Special case for models which are not included in the build list: copy model source files

for %%m in (Align1,Alpha1,Alpha2,CellMM,CloneProbe1,DependencyProbe1,DependencyProbe2,GiniProbe,NewCaseBased_weighted,PA1,THIM,TableProbe1,TableProbe2,TestEx,TypeProbe1) do (

  if not exist %DEPLOY_DIR%\models\%%m (
    call :rcopy_sub_dirs %DEPLOY_DIR%\models models %%m
  )
)

REM OzProjX and OzProjGenX special case:
REM   create models/bin/OzProjX/ompp/bin sub-directory
REM   copy OzProjX*.* into bin/OzProjX/ompp/bin
REM   copy OzProjX/microdata sub-directory into bin/OzProjX

if exist %DEPLOY_DIR%\models\OzProjGenX (
  del /f /q %DEPLOY_DIR%\models\bin\OzProjGenX*.* >> log\build-zip.log 2>&1
  call :make_dir       %DEPLOY_DIR%\models\bin\OzProjGenX\ompp\bin
  call :rcopy_files    %DEPLOY_DIR%\models\bin\OzProjGenX\ompp\bin models\OzProjGenX\ompp\bin "OzProjGenX*.*"
  call :rcopy_sub_dirs %DEPLOY_DIR%\models\bin\OzProjGenX          models\OzProjGenX           "microdata"
  del /f /q            %DEPLOY_DIR%\models\bin\OzProjGenX\ompp\bin\*.log    >> log\build-zip.log 2>&1
  del /f /q            %DEPLOY_DIR%\models\bin\OzProjGenX\ompp\bin\*.tickle >> log\build-zip.log 2>&1
)

if exist %DEPLOY_DIR%\models\OzProjX (
  del /f /q %DEPLOY_DIR%\models\bin\OzProjX*.* >> log\build-zip.log 2>&1
  call :make_dir       %DEPLOY_DIR%\models\bin\OzProjX\ompp\bin
  call :rcopy_files    %DEPLOY_DIR%\models\bin\OzProjX\ompp\bin models\OzProjX\ompp\bin "OzProjX*.*"
  call :rcopy_sub_dirs %DEPLOY_DIR%\models\bin\OzProjX          models\OzProjX          "microdata"
  del /f /q            %DEPLOY_DIR%\models\bin\OzProjX\ompp\bin\*.log    >> log\build-zip.log 2>&1
  del /f /q            %DEPLOY_DIR%\models\bin\OzProjX\ompp\bin\*.tickle >> log\build-zip.log 2>&1
)

REM Copy ompp-docker source code
call :rcopy_files    %DEPLOY_DIR%\ompp-docker ompp-docker "*.*"
call :rcopy_sub_dirs %DEPLOY_DIR%\ompp-docker ompp-docker "debian-build,debian-run,ubuntu-build,ubuntu-run,windows-build,windows-run"

REM Copy ompp-r source code
call :rcopy_files    %DEPLOY_DIR%\ompp-r ompp-r "*.*"
call :rcopy_sub_dirs %DEPLOY_DIR%\ompp-r ompp-r "images,oms-R,openMpp"

REM Copy ompp-python source code
call :rcopy_files    %DEPLOY_DIR%\ompp-python ompp-python "*.*"
call :rcopy_sub_dirs %DEPLOY_DIR%\ompp-python ompp-python "Functions,images,oms-Python"

REM Copy ompp-other source code
call :rcopy_files    %DEPLOY_DIR%\ompp-other ompp-other "*.*"
call :rcopy_sub_dirs %DEPLOY_DIR%\ompp-other ompp-other "azure_cloud,google_cloud,model-build"

REM create zip archive from deployment directory

@echo Build completed on: %DATE% %TIME% > %DEPLOY_DIR%\build_date.txt

@echo Create %DEPLOY_ZIP%
@echo Create %DEPLOY_ZIP% >> log\build-zip.log

C:\7zip\7z.exe a -tzip %DEPLOY_ZIP% %DEPLOY_DIR%\*
if ERRORLEVEL 1 (
  @echo FAIL to create %DEPLOY_ZIP%
  @echo FAIL to create %DEPLOY_ZIP% >> log\build-zip.log
  EXIT
)

@echo %DATE% %TIME% Done.
@echo %DATE% %TIME% Done. >> log\build-zip.log

popd
goto :eof

REM end of main body

REM copy file(s), using copy command
REM arguments:
REM  1 = destination directory
REM  2 = source file(s) path, can be wildcard

:do_copy_files

set dst_dir=%1
set src_path=%2

@echo Copy files from: %src_path%
@echo Copy files from: %src_path% >> log\build-zip.log

copy /b %src_path% %dst_dir% >> log\build-zip-copy.log
if ERRORLEVEL 2 (
  @echo FAIL to copy: %src_path%
  @echo FAIL to copy: %src_path% >> log\build-zip.log
  EXIT
)
exit /b

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
@echo Copy files from: %src_dir% >> log\build-zip.log

robocopy /njh /njs /np %src_dir% %dst_dir% %fnames% >> log\build-zip-copy.log
if ERRORLEVEL 2 (
  @echo FAIL to copy: %fnames%
  @echo FAIL to copy: %fnames% >> log\build-zip.log
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
  @echo Copy directory: %%d >> log\build-zip.log
  
  robocopy /s /e /njh /njs /np %src_root%\%%d %dst_dir%\%%d >> log\build-zip-copy.log
  if ERRORLEVEL 2 (
    @echo FAIL to copy: %%d
    @echo FAIL to copy: %%d >> log\build-zip.log
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
@echo Create directory: %dst_dir% >> log\build-zip.log

mkdir %dst_dir%
if ERRORLEVEL 1 (
  @echo FAIL to create: %dst_dir%
  @echo FAIL to create: %dst_dir% >> log\build-zip.log
  EXIT
)
exit /b

REM if OMPP_BUILD_TAG defined then pushd to git directory and checkout OMPP_BUILD_TAG
REM arguments:
REM  1 = directory of git repository

:do_git_tag_checkout

set dst_dir=%1


if defined OMPP_BUILD_TAG (

  pushd %dst_dir%
  call :do_cmd_line_log ..\log\build-zip.log "git checkout %OMPP_BUILD_TAG%"
  popd
)
exit /b

REM helper subroutine to execute command, log it and check errorlevel
REM arguments:
REM  1 = command line

:do_cmd_line
  call :do_cmd_line_log log\build-zip.log %1
exit /b

REM helper subroutine to execute command, log it and check errorlevel
REM arguments:
REM  1 = log file path
REM  2 = command line

:do_cmd_line_log

set c_log=%1
set c_line=%~2

@echo %c_line%
@echo %c_line% >> %c_log%

%c_line% >> %c_log% 2>&1
if ERRORLEVEL 1 (
  @echo FAILED.
  @echo FAILED. >> %c_log%
  EXIT
) 
exit /b