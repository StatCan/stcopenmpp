@echo off

REM !!! DO NOT use this script if you have multiple models in that database file
REM
REM It does:
REM   rename source modelName.sqlite into modelName-sqlite.db
REM   use sqlite3 to create new empty model db: modelName.db
REM   use dbcopy to copy from old modelName-sqlite.db into new modelName.db
REM   delete old database file modelName-sqlite.db
REM   rename modelName.db into modelName.sqlite
REM
REM Environment:
REM   OM_ROOT   - openM++ root folder, default: ..
REM               must exist:
REM                 OM_ROOT\bin\dbcopy.exe
REM                 OM_ROOT\bin\sqlite3.exe
REM                 OM_ROOT\sql\create_db.sql
REM                 OM_ROOT\sql\insert_default.sql
REM                 OM_ROOT\sql\sqlite\optional_meta_views_sqlite.sql
REM Arguments:
REM        %1 - path to model database
REM        %2 - (optional) model name   (or: no-name), default: database file name
REM        %3 - (optional) model digest (or: no-digest)
REM        %4 - (optional) dbcopy log language (or: und), e.g.: fr-CA

setlocal enabledelayedexpansion

set db_path=%1

IF not [%2] == [] (
  IF not "%2" == "" set m_name=%2
)
IF not [%3] == [] (
  IF not "%3" == "" set m_digest=%3
)
IF not [%4] == [] (
  IF not "%4" == "" set msg_lang=%4
)

IF "%OM_ROOT%" == "" (
  set "OM_ROOT=%~dp0.."
)
@echo OM_ROOT  : %OM_ROOT%

REM check if database file exist

IF "%db_path%" == "" (
  @echo "ERROR: invalid (empty) path to SQLite database file"
  EXIT 1
)
IF not exist "%db_path%" (
  @echo "ERROR: SQLite database file not found (or invalid): %db_path%"
  EXIT 1
)

FOR %%G in ("%db_path%") do set db_file=%%~nxG
FOR %%G in ("%db_path%") do set db_dir=%%~dpG
FOR %%G in ("%db_path%") do set db_stem=%%~nG

@echo Model db : %db_path%

IF "%db_dir%" == "" (
  @echo "ERROR: invalid (empty) model database directory"
  EXIT 1
)

REM if model name not supplied as argument or if it is "no-name"
REM then use db file stem: file name without extension

IF NOT DEFINED m_name (
  set m_name=%db_stem%
) ELSE (
  IF [%m_name%] == [""] set m_name=%db_stem%
  IF "%m_name%" == "no-name" set m_name=%db_stem%
)
IF "%m_name%" == "" (
  @echo "ERROR: invalid model name: %db_stem%"
  EXIT 1
)

@echo Model    : %m_name%

REM model digest is optional or it can be "no-digest"

set m_arg=-m %m_name%

IF DEFINED m_digest (
  IF not [%m_digest%] == [""] (
    IF not "%m_digest%" == "no-digest" (
      set m_arg=-dbcopy.ModelDigest %m_digest%
      @echo Digest   : %m_digest%
    )
  )
)

REM log messages language is optional or it can be "und"

IF DEFINED msg_lang (
  IF not [%msg_lang%] == [""] (
    IF not "%msg_lang%" == "und" (
      set lang_arg=-OpenM.MessageLanguage %msg_lang%
      @echo Language : %msg_lang%
    )
  )
)



REM check OM_ROOT, following must exist:
REM   OM_ROOT/bin/dbcopy.exe
REM   OM_ROOT/bin/sqlite3.exe
REM   OM_ROOT/sql/create_db.sql
REM   OM_ROOT/sql/insert_default.sql
REM   OM_ROOT/sql/sqlite/optional_meta_views_sqlite.sql

set dbcopy_exe=%OM_ROOT%\bin\dbcopy.exe

IF not exist "%dbcopy_exe%" (
  @echo "ERROR: dbcopy utility not found (or invalid): %dbcopy_exe%"
  EXIT 1
)

set SQLITE_EXE=%OM_ROOT%\bin\sqlite3.exe

IF not exist "%SQLITE_EXE%" (
  @echo "ERROR: sqlite3 utility not found: %SQLITE_EXE%"
  EXIT 1
)

IF not exist "%OM_ROOT%\sql\create_db.sql" (
  @echo "ERROR: SQL script not found: %OM_ROOT%\sql\create_db.sql"
  EXIT 1
)
IF not exist "%OM_ROOT%\sql\insert_default.sql" (
  @echo "ERROR: SQL script not found: %OM_ROOT%\sql\insert_default.sql"
  EXIT 1
)
IF not exist "%OM_ROOT%\sql\sqlite\optional_meta_views_sqlite.sql" (
  @echo "ERROR: SQL script not found: %OM_ROOT%\sql\sqlite\optional_meta_views_sqlite.sql"
  EXIT 1
)

REM start database cleanup
REM   rename existing model.sqlite into model-sqlite.db

set src_path=%db_dir%%db_stem%-sqlite.db

rename "%db_path%" "%db_stem%-sqlite.db"

if ERRORLEVEL 1 (
  @echo "ERROR at: rename %db_path% %db_stem%-sqlite.db"
  EXIT
) 

REM create new modelName.db empty file

set new_db=%db_dir%\%db_stem%.db

for /L %%k in (1,1,8) do (
  if exist "%new_db%" (
    del /f /q "%new_db%"
  )
  if exist "%new_db%" (
    ping 127.0.0.1 -n 2 -w 500 >nul
  )
)
if exist "%new_db%" (
  @echo "FAIL to delete: %new_db%"
  EXIT 1
)

call :do_sql_script "%new_db%" "%OM_ROOT%\sql\create_db.sql"
call :do_sql_script "%new_db%" "%OM_ROOT%\sql\insert_default.sql"
call :do_sql_script "%new_db%" "%OM_ROOT%\sql\sqlite\optional_meta_views_sqlite.sql"

REM prepare source database:
REM report run status and lock all worksets

@echo "Source model run status count:"

"%SQLITE_EXE%" "%src_path%" "SELECT status, COUNT(*) FROM run_lst GROUP BY status ORDER BY 1;"

if ERRORLEVEL 1 (
  @echo "ERROR at: SELECT status, COUNT(*) FROM run_lst GROUP BY status ORDER BY 1;"
  EXIT 1
) 

@echo "Update source input scenario: set read-only"

"%SQLITE_EXE%" "%src_path%" "UPDATE workset_lst SET is_readonly = 1;"

if ERRORLEVEL 1 (
  @echo "ERROR at: UPDATE workset_lst SET is_readonly = 1;"
  EXIT 1
) 

@echo "Source input scenario count:"

"%SQLITE_EXE%" "%src_path%" "SELECT COUNT(*) FROM workset_lst;"

if ERRORLEVEL 1 (
  @echo "ERROR at: SELECT COUNT(*) FROM workset_lst;"
  EXIT 1
) 

REM copy model into new database

"%dbcopy_exe%" %m_arg% -dbcopy.To db2db %lang_arg% -dbcopy.ToSqlite "%new_db%" -dbcopy.FromSqlite "%src_path%"

if ERRORLEVEL 1 (
  @echo "ERROR at: %dbcopy_exe% %m_arg% -dbcopy.To db2db %lang_arg% -dbcopy.ToSqlite %new_db% -dbcopy.FromSqlite %src_path%"
  EXIT
)

REM report copy results

@echo "Results model run status:"

"%SQLITE_EXE%" "%new_db%" "SELECT status, COUNT(*) FROM run_lst GROUP BY status ORDER BY 1;"

if ERRORLEVEL 1 (
  @echo "ERROR at: SELECT status, COUNT(*) FROM run_lst GROUP BY status ORDER BY 1;"
  EXIT 1
)

@echo "Results input scenario count:"

"%SQLITE_EXE%" "%new_db%" "SELECT COUNT(*) FROM workset_lst;"

if ERRORLEVEL 1 (
  @echo "ERROR at: SELECT COUNT(*) FROM workset_lst;"
  EXIT 1
)

REM delete old database file
REM rename new database file into modelName.sqlite

for /L %%k in (1,1,8) do (
  if exist "%src_path%" (
    del /f /q "%src_path%"
  )
  if exist "%src_path%" (
    ping 127.0.0.1 -n 2 -w 500 >nul
  )
)
if exist "%src_path%" (
  @echo "FAIL to delete: %src_path%"
  EXIT 1
)

rename "%new_db%" "%db_file%"

if ERRORLEVEL 1 (
  @echo "ERROR at: rename %new_db% %db_file%"
  EXIT 1
)

@echo Done.

goto :eof

REM end of main body

REM do:
REM     sqlite3.exe model.db < script.sql
REM arguments:
REM  1 = path to model database
REM  2 = path to sql script

:do_sql_script

"%SQLITE_EXE%" "%1" < "%2"
if ERRORLEVEL 1 (
  @echo "ERROR at: %SQLITE_EXE% %1 < %2"
  EXIT 1
) 

exit /b
