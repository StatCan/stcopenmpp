#!/usr/bin/env bash
#
# Linux: compress model database by copy the model into a new database
#
# !!! DO NOT use this script if you have multiple models in that database file
#
# It does:
#   rename source modelName.sqlite into modelName-sqlite.db
#   use sqlite3 to create new empty model db: modelName.db
#   use dbcopy to copy from old modelName-sqlite.db into new modelName.db
#   delete old database file modelName-sqlite.db
#   rename modelName.db into modelName.sqlite
#
# Environment:
#   sqlite3   - must be in the $PATH
#   OM_ROOT   - openM++ root folder, default: ..
#               must exist:
#                 $OM_ROOT/bin/dbcopy
#                 $OM_ROOT/sql/create_db.sql
#                 $OM_ROOT/sql/insert_default.sql
#                 $OM_ROOT/sql/sqlite/optional_meta_views_sqlite.sql
# Arguments:
#        $1 - path to model database
#        $2 - (optional) model name   (or: no-name), default: database file name
#        $3 - (optional) model digest (or: no-digest)
#        $4 - (optional) dbcopy log language (or: und), e.g.: fr-CA
#

set -e

self_dir=$(dirname "$0")

db_path="$1"
m_name="$2"
m_digest="$3"
msg_lang="$4"

# check if database file exist

if [ -z "${db_path}" ] ;
then
  echo "ERROR: invalid (empty) path to SQLite database file"
  exit 1
fi
if [ ! -f "${db_path}" ] ;
then
  echo "ERROR: SQLite database file not found (or invalid): ${db_path}"
  exit 1
fi

db_file=$(basename "${db_path}")
db_dir=$(dirname "${db_path}")
db_stem="${db_file%.*}"

echo "Model db : ${db_file}"

if [ -z "${db_dir}" ] ;
then
  echo "ERROR: invalid (empty) model database directory"
  exit 1
fi

# if model name not supplied as argument or if it is "no-name"
# then use db file stem: file name without extension

[ -z "${m_name}" ]          && m_name="${db_stem}"
[ "${m_name}" = "no-name" ] && m_name="${db_stem}"

if [ -z "${m_name}" ] ;
then
  echo "ERROR: invalid model name: ${db_stem}"
  exit 1
fi

echo "Model    : ${m_name}"

# model digest is optional or it can be "no-digest"

m_arg="-m ${m_name}"

if [ -n "${m_digest}" ] && [ "${m_digest}" != "no-digest" ] ;
then
  m_arg="-dbcopy.ModelDigest ${m_digest}"

  echo "Digest   : ${m_digest}"
fi

# log messages language is optional or it can be "und"

lang_arg=

if [ -n "${msg_lang}" ] && [ "${msg_lang}" != "und" ] ;
then
  lang_arg="-OpenM.MessageLanguage ${msg_lang}"
  echo "Language : ${msg_lang}"
fi


# check OM_ROOT, following must exist:
#   $OM_ROOT/bin/dbcopy
#   $OM_ROOT/sql/create_db.sql
#   $OM_ROOT/sql/insert_default.sql
#   $OM_ROOT/sql/sqlite/optional_meta_views_sqlite.sql

[ -z "${OM_ROOT}" ] && OM_ROOT=$(dirname "$self_dir")

echo "OM_ROOT  : ${OM_ROOT}"

dbcopy_exe="${OM_ROOT}"/bin/dbcopy

if [ ! -x "${dbcopy_exe}" ] ;
then
  echo "ERROR: dbcopy utility not found (or invalid): ${dbcopy_exe}"
  exit 1
fi
if [ ! -f "${OM_ROOT}/sql/create_db.sql" ] || [ ! -f "${OM_ROOT}/sql/insert_default.sql" ] || [ ! -f "${OM_ROOT}/sql/sqlite/optional_meta_views_sqlite.sql" ] ;
then
  echo "ERROR: SQL script(s) not found at: ${OM_ROOT}/sql"
  exit 1
fi

# check if sqlite3 exe exists

SQLITE_EXE=sqlite3

if ! command -v "${SQLITE_EXE}" ;
then
  echo "ERROR: $SQLITE_EXE utility not found"
  exit 1
fi

#
# start database cleanup
#   rename existing modelName.sqlite into modelName-sqlite.db
#
do_cmd()
{
  if ! "$@" ;
  then
    echo "ERROR at: $@"
    exit 1
  fi
}
src_path="${db_dir}/${db_stem}-sqlite.db"

do_cmd mv "${db_path}" "${src_path}"

# create new modelName.db empty file

new_db="${db_dir}/${db_stem}.db"

[ -f "${new_db}" ] && do_cmd rm "${new_db}"

do_sql_script()
{
  if ! "${SQLITE_EXE}" "$1" < "$2" ;
  then
    echo "ERROR at: $SQLITE_EXE $1 < $2"
    exit 1
  fi
}

do_sql_script "${new_db}" "${OM_ROOT}/sql/create_db.sql"
do_sql_script "${new_db}" "${OM_ROOT}/sql/insert_default.sql"
do_sql_script "${new_db}" "${OM_ROOT}/sql/sqlite/optional_meta_views_sqlite.sql"

# prepare source database:
# report run status and lock all worksets
#
do_sql_cmd()
{
  if ! echo "$2" | "${SQLITE_EXE}" "$1" ;
  then
    echo "ERROR at: $2"
    exit 1
  fi
}

echo Source model run status count:
do_sql_cmd "${src_path}" "SELECT status, COUNT(*) FROM run_lst GROUP BY status ORDER BY 1;"

echo Update source input scenario: set read-only
do_sql_cmd "${src_path}" "UPDATE workset_lst SET is_readonly = 1;"

echo Source input scenario count:
do_sql_cmd "${src_path}" "SELECT COUNT(*) FROM workset_lst;"

# copy model into new database

do_cmd "${dbcopy_exe}" ${m_arg} -dbcopy.To db2db ${lang_arg} -dbcopy.ToSqlite "${new_db}"  -dbcopy.FromSqlite "${src_path}"

# report copy results

echo Results model run status:
do_sql_cmd "${new_db}" "SELECT status, COUNT(*) FROM run_lst GROUP BY status ORDER BY 1;"

echo Results input scenario count:
do_sql_cmd "${new_db}" "SELECT COUNT(*) FROM workset_lst;"

# delete old database file
# rename new database file into modelName.sqlite

do_cmd rm "${src_path}"
do_cmd mv "${new_db}" "${db_path}"

echo "Done: ${db_path}"
