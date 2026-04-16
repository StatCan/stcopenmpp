@echo off
REM build openM++ Go oms web-service and dbcopy, dbget utilities

setlocal enabledelayedexpansion

REM push into ompp root and make log directory if not exist

if not exist ompp mkdir ompp

if not exist ompp (
  @echo ERROR: missing source code directory: ompp
  EXIT 1
)

pushd ompp

if not exist log mkdir log

REM log build environment 

@echo %DATE% %TIME% Build Go oms web-service and dbcopy, dbget utilities
@echo Log file: log\build-go.log
@echo %DATE% %TIME% Build Go oms web-service and dbcopy, dbget utilities > log\build-go.log

REM build go oms web-service and dbcopy, dbget utilities

if not exist ompp-go (
  @echo ERROR: missing source code directory: go
  EXIT 1
)


pushd ompp-go

REM do build

call :do_cmd_line "go install -tags odbc,sqlite_math_functions,sqlite_omit_load_extension ./dbcopy"
call :do_cmd_line "go install -tags odbc,sqlite_math_functions,sqlite_omit_load_extension ./dbget"
call :do_cmd_line "go install -tags odbc,sqlite_math_functions,sqlite_omit_load_extension ./oms"

popd

@echo %DATE% %TIME% Done.
@echo %DATE% %TIME% Done. >> log\build-go.log

popd
goto :eof

REM end of main body

REM helper subroutine to execute command, log it and check errorlevel
REM arguments:
REM  1 = command line

:do_cmd_line
  call :do_cmd_line_log ..\log\build-go.log %1
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
