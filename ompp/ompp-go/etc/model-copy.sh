#!/usr/bin/env bash
#
# Copy model files from source $src_root to destination $dst_root, for example:
#
# model-copy.sh RiskPaths.publish.lst ~/archive ~/my-work RiskPaths v3.2.1
#
# Arguments:
#
#   $1 - publish_lst: path to model publish list file, if relative then must be relative to source $src_root
#   $2 - src_root   : source root path, if relative then must be relative to OM_ROOT
#   $3 - dst_root   : destination root, if relative then must be relative to OM_ROOT
#   $4 - name_ver   : model name or name-version e.g.: RiskPaths-v3.2.1
#
# It does:
#   - reads list of model files from $publish_lst, e.g.: from RiskPaths.publish.lst
#   - create destination directories (if not exists) and copy files
#   - save list of files into model copy list, e.g.: RiskPaths-v3.2.1.copy.lst
#
# Environment:
#
#   OM_ROOT                       (optional) openM++ root path
#   BIN_DIR  default: models/bin  sub-folder where model.exe and model.sqlite resides
#   DOC_DIR  default: models/doc  models documentation sub-folder
#   LOG_DIR  default: models/log  models log sub-folder
#
# Example:
#
# model-copy.sh models/bin/RiskPaths.publish.lst ~/archive ~/my-work RiskPaths
#
# where ~/archive/models/bin/RiskPaths.publish.lst :
#
#   $BIN_DIR/RiskPaths
#   $BIN_DIR/RiskPaths.sqlite
#   $DOC_DIR/RiskPaths.doc.EN.html
#   $DOC_DIR/RiskPaths.doc.FR.html
#   $LOG_DIR/RiskPaths.log
#   some_data.file
#
# It does copy from => to:
#
# ~/archive/models/bin/RiskPaths             => ~/my-work/models/bin/RiskPaths
# ~/archive/models/bin/RiskPaths.sqlite      => ~/my-work/models/bin/RiskPaths.sqlite
# ~/archive/models/doc/RiskPaths.doc.EN.html => ~/my-work/models/doc/RiskPaths.doc.EN.html
# ~/archive/models/doc/RiskPaths.doc.FR.html => ~/my-work/models/doc/RiskPaths.doc.FR.html
# ~/archive/models/log/RiskPaths.log         => ~/my-work/models/log/RiskPaths.log
# ~/archive/some_data.file                   => ~/my-work/some_data.file
#

set -e

publish_lst="$1"
src_root="$2"
dst_root="$3"
name_ver="$4"

if [ -z "$publish_lst" ] ;
then
  echo "ERROR: invalid (empty) path to model files list"
  exit 1
fi
if [ -z "$src_root" ] ;
then
  echo "ERROR: invalid (empty) source directory"
  exit 1
fi
if [ -z "$dst_root" ] ;
then
  echo "ERROR: invalid (empty) destination directory"
  exit 1
fi
if [ -z "$name_ver" ] ;
then
  echo "ERROR: invalid (empty) model name-version"
  exit 1
fi

# set model files sub-directories, if not defined then use defaults

[ -z "$BIN_DIR" ] && BIN_DIR="models/bin"
[ -z "$DOC_DIR" ] && DOC_DIR="models/doc"
[ -z "$LOG_DIR" ] && LOG_DIR="models/log"

echo "Model   : $name_ver"
echo "Copy    : $publish_lst"
echo "From    : $src_root"
echo "To      : $dst_root"
echo "OM_ROOT : $OM_ROOT"
echo "BIN_DIR : $BIN_DIR"
echo "DOC_DIR : $DOC_DIR"
echo "LOG_DIR : $LOG_DIR"

# make absolute directory path if directory path is relative to $PWD
# it removes last / or last /.
# it does not normalize path and it does not check if path exists
#
# openmpp/mdls/    => /home/user/openmpp/mdls
# openmpp/mdls/.   => /home/user/openmpp/mdls
# openmpp/mdls/..  => /home/user/openmpp/mdls/..
# .                => /home/user
# ..               => /home/user/..
# /openmpp/mdls/   => /openmpp/mdls
# /openmpp/mdls/.  => /openmpp/mdls
# /openmpp/mdls/.. => /openmpp/mdls/..
# /                => empty result
# /.               => empty result
#
do_abspath()
{
  wd="$PWD"
 [ "$PWD" = "/" ] && wd=""

  case "$1" in
    /*) d="${1}"
    ;;
    *)  d="$wd/${1#./}"
    ;;
  esac

  d="${d%/}"
  d="${d%/.}"

  echo "$d"
}

# execute command and exit if failed

do_cmd()
{
  if ! "$@" ;
  then
    echo "ERROR at: $@"
    exit 1
  fi
}

# if defined OM_ROOT then pushd to OM_ROOT

if [ -n "$OM_ROOT" ] ;
then

  if [ ! -d "$OM_ROOT" ] ;
  then
    echo "ERROR: invalid directory: $OM_ROOT"
    exit 1
  fi
  OM_ROOT=$(do_abspath "$OM_ROOT")

  do_cmd pushd "$OM_ROOT"
fi

# make absolute path to source snd destination directory

abs_src_root=$(do_abspath "$src_root")
abs_dst_root=$(do_abspath "$dst_root")

if [ -z "$abs_src_root" ] || [ "$abs_src_root" = "/" ] ;
then
  echo "ERROR: invalid (empty) source directory"
  exit 1
fi
if [ -z "$abs_dst_root" ] || [ "$abs_dst_root" = "/" ] ;
then
  echo "ERROR: invalid (empty) destination directory"
  exit 1
fi
if [ "$abs_src_root" = "$abs_dst_root" ] ;
then
  echo "ERROR: source and destination directory are the same: $abs_src_root"
  exit 1
fi

# check if source root exist

if [ ! -d "$src_root" ] ;
then
  echo "ERROR: invalid source directory: $src_root"
  exit 1
fi

# check if publish list file exist in $src_root and get absolute path to $publish_lst

do_cmd pushd "$abs_src_root"

abs_pub_lst=$(do_abspath "$publish_lst")

if [ -z "$abs_pub_lst" ] ;
then
  echo "ERROR: invalid (or empty) absolute path to model files list"
  exit 1
fi

popd  # popd from $abs_src_root to $OM_ROOT

if [ ! -f "$abs_pub_lst" ] ;
then
  echo "ERROR: to model publish list file not found: $abs_pub_lst"
  exit 1
fi

# create destination root if not exist and pushd into destination directory

if [ ! -d "$dst_root" ] ;
then
  do_cmd mkdir -p "$dst_root"
fi

do_cmd pushd "$dst_root"

# read from $publish_lst
# replace $BIN_DIR $DOC_DIR $LOG_DIR with actual path
# copy eachfile or directry from $src_root to $dst_root

# append list of copied items into $dst_root/$BIN_DIR/ModelName-version.copy.lst

abs_cp_lst="$abs_dst_root/${BIN_DIR}/${name_ver}.copy.lst"

if [ -f "$abs_cp_lst" ] ;
then
  do_cmd rm "$abs_cp_lst"
fi

while IFS=$' \t\r\n' read -r r_path || [ -n "$r_path" ]; do

  # replace $BIN_DIR $DOC_DIR $LOG_DIR with actual path

  r_path=${r_path/\$BIN_DIR/"$BIN_DIR"}
  r_path=${r_path/\$DOC_DIR/"$DOC_DIR"}
  r_path=${r_path/\$LOG_DIR/"$LOG_DIR"}

  [ -z "$r_path" ] && continue # skip empty source line

  # relative path must be inside of $src_root and $dst_root, it cannot go up ../

  case "$r_path" in
    *".."*)
      echo "ERROR: invalid path: $r_path it cannot contain .."
      exit 1
    ;;
  esac

  a_f_path="$abs_src_root/$r_path"
  a_t_path="$abs_dst_root/$r_path"

  if [ ! -e "$a_f_path" ] ;
  then
    echo "ERROR: source file or directory not found: $a_f_path"
    exit 1
  fi
  
  # if source is a file path
  #    then create destination directory
  #    and copy the file
  # if source is a directory path
  #    then create destination directory
  #    and if source directry not empty then copy it content recursively

  if [ -d "$a_f_path" ] ;
  then
    [ ! -d "$a_t_path" ] && do_cmd mkdir -p "$a_t_path"

    cnt=$(ls -1 "$a_f_path"| wc -l)

    [ "$cnt" != "0" ] && do_cmd cp -pr $a_f_path/* "$a_t_path"

  else

    d=$(dirname "$a_t_path")
    [ ! -d "$d" ] && do_cmd mkdir -p "$d"

    do_cmd cp -p "$a_f_path" "$d"
  fi

  printf "%s\n" "$a_t_path" >> "$abs_cp_lst"

done < "$abs_pub_lst"

