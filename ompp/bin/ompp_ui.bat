@echo off
setlocal

rem Default OM_ROOT to the parent of this script's folder, as a clean absolute path
if "%OM_ROOT%" == "" (
  for %%I in ("%~dp0..") do set "OM_ROOT=%%~fI"
)
if "%OMS_PORT%" == "" set "OMS_PORT=4040"

echo OM_ROOT: "%OM_ROOT%"

if not exist "%OM_ROOT%\bin\oms.exe" (
  echo ERROR: missing oms executable: "%OM_ROOT%\bin\oms.exe"
  goto :fail
)

if not exist "%OM_ROOT%\models\bin" (
  echo ERROR: missing models bin directory: "%OM_ROOT%\models\bin"
  goto :fail
)

if not exist "%OM_ROOT%\html" (
  echo ERROR: missing UI html directory: "%OM_ROOT%\html"
  goto :fail
)

if not exist "%OM_ROOT%\log" mkdir "%OM_ROOT%\log"

set "OM_CFG_INI_ALLOW=true"
set "OM_CFG_INI_ANY_KEY=true"
if "%OM_CFG_TYPE_MAX_LEN%" == "" set "OM_CFG_TYPE_MAX_LEN=366"

start "oms" /D "%OM_ROOT%" "%OM_ROOT%\bin\oms.exe" ^
  -oms.RootDir "%OM_ROOT%" ^
  -oms.HomeDir models\home ^
  -oms.Listen localhost:%OMS_PORT% ^
  -oms.AllowDownload ^
  -oms.AllowUpload ^
  -oms.AllowMicrodata ^
  -oms.PidSaveTo "%OM_ROOT%\log\oms.pid.txt" ^
  -oms.LogRequest ^
  -OpenM.LogFilePath "%OM_ROOT%\log\oms.log"

rem Give the server a moment to start before opening the browser
timeout /t 2 /nobreak >nul
start "" "http://localhost:%OMS_PORT%"

exit /b 0

:fail
pause
exit /b 1