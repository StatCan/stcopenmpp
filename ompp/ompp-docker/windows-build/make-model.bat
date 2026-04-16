@echo off
REM build and run openM++ model

REM environmemnt variables:
REM
REM  set OM_ROOT                   default: ..\..
REM  set OM_BUILD_CONFIGS=Debug    default: Release
REM  set OM_BUILD_PLATFORMS=Win32  default: x64
REM  set OM_MSG_USE=MPI            default: EMPTY
REM  set MODEL_NAME           if not empty then model name
REM  set MODEL_DIR            if not empty then source code model directory
REM  set MODEL_GIT_URL        if not empty then git URL of model source code
REM  set MODEL_GIT_TAG        if not empty then git tag
REM  set MODEL_INI            if not empty then run model after build with this model ini file
REM  set MODEL_DOC_DISABLE=0  if =0 or =false then make model documentation, default: do not make model documemtation

REM one of: MODEL_NAME or MODEL_DIR must be defined
REM
REM if MODEL_NAME not empty and MODEL_DIR empty then MODEL_DIR  = OM_ROOT\models\MODEL_NAME
REM if MODEL_NAME empty and MODEL_DIR not empty then MODEL_NAME = last element of MODEL_DIR

REM if MODEL_DIR not exists and MODEL_GIT_URL specified then do git clone model source code

setlocal enabledelayedexpansion

IF "%OM_ROOT%" == "" (
  set "OM_ROOT=%~dp0..\.."
)
@echo OM_ROOT        = %OM_ROOT%

if not exist "%OM_ROOT%\bin\omc.exe" (
  @echo ERROR: invalid OM_ROOT = %OM_ROOT%
  EXIT 1
)

REM one of: MODEL_NAME or MODEL_DIR must be defined
REM
REM if MODEL_NAME not empty and MODEL_DIR empty then MODEL_DIR  = OM_ROOT\models\MODEL_NAME
REM if MODEL_NAME empty and MODEL_DIR not empty then MODEL_NAME = last element of MODEL_DIR

if "%MODEL_NAME%" == "" (
  if "%MODEL_DIR%" == "" (
    @echo ERROR: both MODEL_NAME and MODEL_DIR are not defined
    EXIT 1
  )
)

if not "%MODEL_NAME%" == "" (
  if "%MODEL_DIR%" == "" set MODEL_DIR=%OM_ROOT%\models\%MODEL_NAME%
  set "OM_P_MDL_NAME=-p:MODEL_NAME=%MODEL_NAME%"
)

if "%MODEL_DIR:~-1%"=="\" set "MODEL_DIR=%MODEL_DIR:~0,-1%"

if "%MODEL_NAME%" == "" (

 for %%D in ("%MODEL_DIR%") do set mdl_dir=%%~nxD
 if "%MODEL_NAME%" == "" set MODEL_NAME=!mdl_dir!
)

if "%MODEL_NAME%" == "" (
    @echo ERROR: empty MODEL_NAME
    EXIT 1
)
if "%MODEL_DIR%" == "" (
  @echo ERROR: empty MODEL_DIR
  EXIT 1
)

REM set default build configuration

set OM_BLD_CFG=Release
set OM_BLD_PLT=x64

if defined OM_BUILD_CONFIGS   set OM_BLD_CFG=%OM_BUILD_CONFIGS%
if defined OM_BUILD_PLATFORMS set OM_BLD_PLT=%OM_BUILD_PLATFORMS%
if /I "%OM_MSG_USE%"=="MPI"   set OM_P_MPI=-p:OM_MSG_USE=MPI


set OM_P_DOC=-p:MODEL_DOC=false

if /I "%MODEL_DOC_DISABLE%"=="0"     set "OM_P_DOC="
if /I "%MODEL_DOC_DISABLE%"=="false" set "OM_P_DOC="

REM log build environment 

if not exist "%OM_ROOT%\log" mkdir "%OM_ROOT%\log"
set LOG_PATH="%OM_ROOT%\log\make-%MODEL_NAME%.log"

@echo %DATE% %TIME% Make %MODEL_NAME% model
@echo OM_ROOT            = %OM_ROOT%
@echo MODEL_NAME         = %MODEL_NAME%
@echo MODEL_DIR          = %MODEL_DIR%
@echo OM_BUILD_CONFIGS   = %OM_BLD_CFG%
@echo OM_BUILD_PLATFORMS = %OM_BLD_PLT%
@echo OM_MSG_USE         = %OM_MSG_USE%
@echo MODEL_GIT_URL      = %MODEL_GIT_URL%
@echo MODEL_GIT_TAG      = %MODEL_GIT_TAG%
@echo MODEL_INI          = %MODEL_INI%
if defined OM_P_MPI (
  @echo Make cluster version: using MPI
) else (
  @echo Make desktop version: non-MPI
)
if not defined OM_P_DOC (
  @echo Make model documentation
)

@echo Log file: %LOG_PATH%

@echo %DATE% %TIME% Make %MODEL_NAME% model > "%LOG_PATH%"
@echo OM_ROOT            = %OM_ROOT% >> "%LOG_PATH%"
@echo MODEL_NAME         = %MODEL_NAME% >> "%LOG_PATH%"
@echo MODEL_DIR          = %MODEL_DIR% >> "%LOG_PATH%"
@echo OM_BUILD_CONFIGS   = %OM_BLD_CFG% >> "%LOG_PATH%"
@echo OM_BUILD_PLATFORMS = %OM_BLD_PLT% >> "%LOG_PATH%"
@echo OM_MSG_USE         = %OM_MSG_USE% >> "%LOG_PATH%"
@echo MODEL_GIT_URL      = %MODEL_GIT_URL% >> "%LOG_PATH%"
@echo MODEL_GIT_TAG      = %MODEL_GIT_TAG% >> "%LOG_PATH%"
@echo MODEL_INI          = %MODEL_INI% >> "%LOG_PATH%"
if defined OM_P_MPI (
  @echo Make cluster version: using MPI >> "%LOG_PATH%"
) else (
  @echo Make desktop version: non-MPI >> "%LOG_PATH%"
)
if not defined OM_P_DOC (
  @echo Make model documentation >> "%LOG_PATH%"
)

REM if MODEL_DIR not exists and MODEL_GIT_URL specified then do git clone model source code

if not "%MODEL_GIT_URL%" == "" (
  if exist "%MODEL_DIR%" (

  @echo Skip: git clone

  ) else (

    @echo git clone %MODEL_GIT_URL% "%MODEL_DIR%"
    git clone %MODEL_GIT_URL% "%MODEL_DIR%" >> "%LOG_PATH%" 2>&1
    if ERRORLEVEL 1 (
      @echo FAILED.
      @echo FAILED: git clone %MODEL_GIT_URL% "%MODEL_DIR%" >> "%LOG_PATH%"
      EXIT
    )

    REM fix git clone issue:
    REM ....fatal: detected dubious ownership in repository at 'C:/build/ompp'

    @echo git config --global --add safe.directory *

    git config --global --add safe.directory * >> "%LOG_PATH%" 2>&1
    if ERRORLEVEL 1 (
      @echo FAILED.
      EXIT
    )

    if defined MODEL_GIT_TAG (

      @echo  MODEL_GIT_TAG = %MODEL_GIT_TAG%

      @echo pushd "%MODEL_DIR%"
      @echo pushd "%MODEL_DIR%" >>  "%LOG_PATH%"

      pushd "%MODEL_DIR%"
      if ERRORLEVEL 1 (
        @echo FAILED: pushd "%MODEL_DIR%" >> "%LOG_PATH%"
        @echo FAILED.
        EXIT
      )

      @echo git checkout %MODEL_GIT_TAG%
      @echo git checkout %MODEL_GIT_TAG% >> "%LOG_PATH%"

      git checkout %MODEL_GIT_TAG% >> "%LOG_PATH%" 2>&1
      if ERRORLEVEL 1 (
        @echo FAILED: git checkout %MODEL_GIT_TAG% >> "%LOG_PATH%"
        @echo FAILED.
        EXIT
      )

      @echo popd
      @echo popd >>  "%LOG_PATH%"

      popd
    )
  )
)

REM check if model source code directory exist

if not exist "%MODEL_DIR%" (
  @echo ERROR: missing source code directory: %MODEL_DIR%
  @echo ERROR: missing source code directory: %MODEL_DIR% >> "%LOG_PATH%"
  EXIT 1
)

REM build model

set MDL_EXE=%MODEL_NAME%
if /i "%OM_BLD_CFG%"=="Debug" set MDL_EXE=%MODEL_NAME%D
if defined OM_P_MPI set MDL_EXE=%MDL_EXE%_mpi
     
set MDL_P_ARGS=-p:Configuration=%OM_BLD_CFG% -p:Platform=%OM_BLD_PLT% %OM_P_MDL_NAME% %OM_P_DOC% %MODEL_NAME%-ompp.sln

call :make_model_sln %MODEL_DIR% %OM_ROOT% %LOG_PATH% "%OM_P_MPI% %MDL_P_ARGS%"
      
REM run the model if model ini-file specified
      
if not "%MODEL_INI%" == "" call :model_run %MODEL_DIR%\ompp\bin "%MODEL_INI%" "%LOG_PATH%"


@echo %DATE% %TIME% Done.
@echo %DATE% %TIME% Done. >>  "%LOG_PATH%"

popd
goto :eof

REM end of main body

REM build model subroutine
REM arguments:
REM  1 = model directory
REM  2 = OM_ROOT
REM  3 = model build log file name
REM  4 = msbuild command line arguments

:make_model_sln

set m_dir=%1
set om_rt=%~f2
set m_log=%~f3
set mk_args=%~4

@echo pushd %m_dir%
@echo pushd %m_dir% >> %m_log%

pushd %m_dir%

setlocal enabledelayedexpansion
set OM_ROOT=%om_rt%

@echo msbuild %mk_args%
@echo msbuild %mk_args% >> %m_log%

msbuild %mk_args% >>%m_log% 2>&1
if ERRORLEVEL 1 (
  @echo FAILED.
  @echo FAILED. >> %m_log%
  EXIT
)
endlocal
popd

exit /b


REM build model subroutine
REM arguments:
REM  1 = model exe directory: MODEL_DIR\ompp\bin
REM  2 = path to model ini file
REM  3 = model build log file name

:model_run

set m_bin=%~f1
set m_ini=%~f2
set m_log=%~f3

@echo pushd %m_bin%
@echo pushd %m_bin% >> %m_log%

pushd %m_bin%

@echo Run: %MDL_EXE% -ini %m_ini%
@echo Run: %MDL_EXE% -ini %m_ini% >> "%m_log%"

%MDL_EXE% -ini "%m_ini%" >> "%m_log%" 2>&1
if ERRORLEVEL 1 (
  @echo FAILED.
  @echo FAILED. >> "%m_log%"
  EXIT
)
popd
exit /b
