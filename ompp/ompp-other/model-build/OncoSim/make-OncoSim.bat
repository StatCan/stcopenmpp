@echo off
REM build and run OncoSim model

REM environmemnt variables:
REM
REM  set OM_ROOT                   default: ..\..
REM  set OM_BUILD_CONFIGS=Debug    default: Release
REM  set OM_BUILD_PLATFORMS=Win32  default: x64
REM  set OM_MSG_USE=MPI            default: EMPTY
REM  set MODEL_DIR            if not empty then source code model directory
REM  set MODEL_GIT_URL        if not empty then git URL of model source code
REM  set MODEL_GIT_TAG        if not empty then git tag
REM  set MODEL_INI            if not empty then run model after build with this model ini file
REM  set MODEL_DOC_DISABLE=0  if =0 or =false then make model documentation, default: do not make model documemtation

REM  set ONCOSIM_VARIANT      if not empty then model variant: breast gmm cervical colorectal lung allcancers
REM  set ONCOSIM_SLN          solution name, default: OncoSimX-ompp.sln

REM MODEL_NAME by default: OncoSim or OncoSim-%ONCOSIM_VARIANT%
REM MODEL_DIR  by default: OM_ROOT\models\OncoSim

REM if MODEL_DIR not exists and MODEL_GIT_URL specified then do git clone model source code
REM if MODEL_DIR exists then it must contain code\src-OncoSimX_variants.ompp.txt == copy of original code\OncoSimX_variants.ompp

setlocal enabledelayedexpansion

IF "%OM_ROOT%" == "" (
  set "OM_ROOT=%~dp0..\.."
)
@echo OM_ROOT        = %OM_ROOT%

if not exist "%OM_ROOT%\bin\omc.exe" (
  @echo ERROR: invalid OM_ROOT = %OM_ROOT%
  EXIT 1
)

REM ONCOSIM_VARIANT if not empty must be one of: breast gmm cervical colorectal lung allcancers

set ONCO_VARIANT=

if defined ONCOSIM_VARIANT (

  if /I "%ONCOSIM_VARIANT%"=="breast" set ONCO_VARIANT=breast
  if /I "%ONCOSIM_VARIANT%"=="gmm" set ONCO_VARIANT=gmm
  if /I "%ONCOSIM_VARIANT%"=="cervical" set ONCO_VARIANT=cervical
  if /I "%ONCOSIM_VARIANT%"=="colorectal" set ONCO_VARIANT=colorectal
  if /I "%ONCOSIM_VARIANT%"=="lung" set ONCO_VARIANT=lung
  if /I "%ONCOSIM_VARIANT%"=="allcancers" set ONCO_VARIANT=allcancers
  
  if not defined ONCO_VARIANT (
    @echo ERROR: invalid ONCOSIM_VARIANT = %ONCOSIM_VARIANT%
    EXIT 1
  )
)

REM MODEL_NAME by default: OncoSim or OncoSim-%ONCOSIM_VARIANT%
REM MODEL_DIR  by default: OM_ROOT\models\OncoSim

if not defined MODEL_NAME (
  set MODEL_NAME=OncoSim
  if defined ONCO_VARIANT set MODEL_NAME=OncoSim-%ONCO_VARIANT%
)

set "OM_P_MDL_NAME=-p:MODEL_NAME=%MODEL_NAME%"

if "%MODEL_DIR%" == "" set MODEL_DIR=%OM_ROOT%\models\OncoSim

if "%MODEL_DIR:~-1%"=="\" set "MODEL_DIR=%MODEL_DIR:~0,-1%"

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

set ONCO_SLN=OncoSimX-ompp.sln
if defined ONCOSIM_SLN set ONCO_SLN=%ONCOSIM_SLN%

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
@echo ONCOSIM_VARIANT    = %ONCO_VARIANT%
@echo ONCOSIM_SLN        = %ONCO_SLN%
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
@echo ONCOSIM_VARIANT    = %ONCO_VARIANT% >> "%LOG_PATH%"
@echo ONCOSIM_SLN        = %ONCO_SLN% >> "%LOG_PATH%"
if defined OM_P_MPI (
  @echo Make cluster version: using MPI >> "%LOG_PATH%"
) else (
  @echo Make desktop version: non-MPI >> "%LOG_PATH%"
)
if not defined OM_P_DOC (
  @echo Make model documentation >> "%LOG_PATH%"
)

REM if MODEL_DIR not exists and MODEL_GIT_URL specified then do git clone model source code
REM copy OncoSimX_variants.ompp => src-OncoSimX_variants.ompp.txt

if not "%MODEL_GIT_URL%" == "" (
  if exist "%MODEL_DIR%" (

  @echo Skip: git clone
  @echo Skip: git clone >> "%LOG_PATH%"

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

    @echo pushd "%MODEL_DIR%"
    @echo pushd "%MODEL_DIR%" >>  "%LOG_PATH%"

    pushd "%MODEL_DIR%"
    if ERRORLEVEL 1 (
      @echo FAILED: pushd "%MODEL_DIR%" >> "%LOG_PATH%"
      @echo FAILED.
      EXIT
    )

    if defined MODEL_GIT_TAG (

      @echo  MODEL_GIT_TAG = %MODEL_GIT_TAG%

      @echo git checkout %MODEL_GIT_TAG%
      @echo git checkout %MODEL_GIT_TAG% >> "%LOG_PATH%"

      git checkout %MODEL_GIT_TAG% >> "%LOG_PATH%" 2>&1
      if ERRORLEVEL 1 (
        @echo FAILED: git checkout %MODEL_GIT_TAG% >> "%LOG_PATH%"
        @echo FAILED.
        EXIT
      )
    )

    REM copy OncoSimX_variants.ompp => src-OncoSimX_variants.ompp.txt
    
    @echo copy code\OncoSimX_variants.ompp code\src-OncoSimX_variants.ompp.txt
    @echo copy code\OncoSimX_variants.ompp code\src-OncoSimX_variants.ompp.txt >> "%LOG_PATH%"

    copy code\OncoSimX_variants.ompp code\src-OncoSimX_variants.ompp.txt
    if ERRORLEVEL 1 (
      @echo FAILED copy OncoSimX_variants.ompp
      @echo FAILED.
      EXIT
    )

    @echo popd
    @echo popd >>  "%LOG_PATH%"

    popd
  )
)

REM check if model source code directory exist

if not exist "%MODEL_DIR%" (
  @echo ERROR: missing source code directory: %MODEL_DIR%
  @echo ERROR: missing source code directory: %MODEL_DIR% >> "%LOG_PATH%"
  EXIT 1
)

REM make source files to build OncoSim variant

if defined ONCO_VARIANT call :prepare_variant "%MODEL_DIR%" %ONCO_VARIANT% "%LOG_PATH%"

REM build model

set MDL_EXE=%MODEL_NAME%
if /i "%OM_BLD_CFG%"=="Debug" set MDL_EXE=%MODEL_NAME%D
if defined OM_P_MPI set MDL_EXE=%MDL_EXE%_mpi
     
set MDL_P_ARGS=-p:Configuration=%OM_BLD_CFG% -p:Platform=%OM_BLD_PLT% %OM_P_MDL_NAME% %OM_P_DOC% OncoSimX-ompp.sln

call :make_model_sln "%MODEL_DIR%" "%OM_ROOT%" "%LOG_PATH%" "%OM_P_MPI% %MDL_P_ARGS%"
      
REM run the model if model ini-file specified
      
if not "%MODEL_INI%" == "" call :model_run %MODEL_DIR%\ompp\bin "%MODEL_INI%" "%LOG_PATH%"


@echo %DATE% %TIME% Done.
@echo %DATE% %TIME% Done. >> "%LOG_PATH%"

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
set m_log=%3
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
set m_log=%3

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

REM prepare OncoSim variant source files:
REM OncoSimX_variants.ompp and CancerModeChoice.dat
REM arguments:
REM arguments:
REM  1 = model directory
REM  2 = model variant
REM  3 = model build log file name

:prepare_variant

set m_dir=%~f1
set m_var=%2
set m_log=%3

setlocal enabledelayedexpansion

  @echo Prepare OncoSim %m_var%
  @echo Prepare OncoSim %m_var% >> %m_log%

  @echo pushd %m_dir%
  @echo pushd %m_dir% >> %m_log%

  pushd %m_dir%

  REM check: if file exist MODEL_DIR\code\src-OncoSimX_variants.ompp.txt
  
  set "src_var_ompp=code\src-OncoSimX_variants.ompp.txt"
  set "dst_var_ompp=code\OncoSimX_variants.ompp"
  
  if not exist "!src_var_ompp!" (
    @echo ERROR: not found: !src_var_ompp!
    @echo ERROR: not found: !src_var_ompp! >> %m_log%
    EXIT 1
  )

  REM uncomment cancer variant by copy from src-OncoSimX_variants.ompp.txt into OncoSimX_variants.ompp

  set v_begin=//begin-%m_var%
  set v_end=//end-%m_var%

  set /A n_begin=0
  set /A n_end=0

  REM Make empty OncoSimX_variants.ompp

  @echo "copy /y NUL !dst_var_ompp!"
  @echo "copy /y NUL !dst_var_ompp!" >> %m_log%
  
  copy /y NUL !dst_var_ompp! >NUL
  if ERRORLEVEL 1 (
    @echo FAILED !dst_var_ompp!
    @echo FAILED.
    EXIT
  )

  set /A n_line=0

  for /f "tokens=1* delims=:" %%a in ('findstr /n .* "!src_var_ompp!"') do (
    set /A "n_line+=1"
    set "line_txt=%%b"

    rem check if begin-variant found
    if defined line_txt (
      if /i not "!line_txt:%v_begin%=!"=="!line_txt!" (
        set /A "n_begin=!n_line!"
      )
    )

    rem for lines before begin-variant or after end-variant output copy it as is
    if !n_begin! LEQ 0 (
      @echo.!line_txt!>>!dst_var_ompp!
    ) else (
      if !n_end! GTR 0 (
        @echo.!line_txt!>>!dst_var_ompp!
      )
    )

    rem for lines between begin-variant and end-variant uncomment // the line
    if !n_begin! GTR 0 (
      if !n_end! LEQ 0 (
        for /f "usebackq tokens=* eol=`" %%m in ('!line_txt!') do (

          set "c_line=%%m"
          if "!c_line:~0,2!"=="//" (
            set "c_line=!c_line:~2%!"
          )
          @echo.!c_line!>>!dst_var_ompp!
        )
      )
    )

    rem check if end-variant found
    if defined line_txt (
      if !n_begin! GTR 0 (
        if /i not "!line_txt:%v_end%=!"=="!line_txt!" (
          set /A "n_end=!n_line!"
        )
      )
    )
  )

  @echo %v_begin% : %n_begin%
  @echo %v_begin% : %n_begin% >> %m_log%
  @echo %v_end% : %n_end%
  @echo %v_end% : %n_end% >> %m_log%

  if %n_begin% LEQ 0 (
    @echo ERROR: MISSING %v_begin%
    @echo ERROR: MISSING %v_begin% >> %m_log%
    EXIT 1
  ) else (
    if %n_end% LEQ 0 (
      @echo ERROR: MISSING %v_end%
      @echo ERROR: MISSING %v_end% >> %m_log%
      EXIT 1
    )
  )

  REM copy CancerModeChoice for that variant
  
  set onco_cm_src=CancerModeChoice.dat.INDEPTH.txt
  if "%m_var%" == "allcancers" set onco_cm_src=CancerModeChoice.dat.ALLCANCERS.txt
  
  @echo copy parameters\Default\%onco_cm_src% parameters\Default\CancerModeChoice.dat
  @echo copy parameters\Default\%onco_cm_src% parameters\Default\CancerModeChoice.dat >> %m_log%

  copy parameters\Default\%onco_cm_src% parameters\Default\CancerModeChoice.dat
  if ERRORLEVEL 1 (
    @echo FAILED copy into CancerModeChoice.dat
    @echo FAILED.
    EXIT
  )

  @echo popd
  @echo popd >>  "%LOG_PATH%"

  popd

endlocal
exit /b
