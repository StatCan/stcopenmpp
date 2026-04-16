## Using openmpp-build image to build and run OncoSim model variants

To build and (optionally) OncoSim model do:

docker run .... openmpp/openmpp-build:debian ./make-OncoSim

Debian Linux Docker examples:

docker run \
  -v $HOME/stable-debian/build:/home/build \
  -e OMPP_USER=build -e OMPP_GROUP=build -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
  -e OM_ROOT=ompp \
  -e MODEL_DIR=ompp/models/OncoSim \
  -e OMC_CODE_PAGE=ISO-8859-15 \
  openmpp/openmpp-build:debian \
  ./make-OncoSim

docker run \
  -v $HOME/stable-debian/build:/home/build \
  -e OMPP_USER=build -e OMPP_GROUP=build -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
  -e OM_ROOT=openmpp_debian_20250601 \
  -e MODEL_NAME=OncoSim \
  -e OMC_CODE_PAGE=ISO-8859-15 \
  -e MODEL_INI=my/OncoSim-test.ini \
  openmpp/openmpp-build:debian \
  ./make-OncoSim

docker run \
  -v $HOME/stable-debian/onco:/home/build \
  -e OMPP_USER=build -e OMPP_GROUP=build -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
  -e OM_ROOT=openmpp_debian_mpi_20250601 \
  -e MODEL_DIR=OncoSim \
  -e OM_MSG_USE=MPI \
  -e MODEL_DOC_DISABLE=0 \
  -e OMC_CODE_PAGE=ISO-8859-15 \
  -e ONCOSIM_VARIANT=lung \
  openmpp/openmpp-build:debian \
  ./make-OncoSim

docker run \
  -v $HOME/stable-debian/build:/home/build \
  -e OMPP_USER=build -e OMPP_GROUP=build -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
  -e OM_ROOT=ompp \
  -e MODEL_NAME=OncoSim \
  -e MODEL_DIR=my/OncoSim \
  -e OMC_CODE_PAGE=ISO-8859-15 \
  -e MODEL_GIT_URL=https://gitlab.com/my/OncoSim.git
  -e MODEL_GIT_TAG=v1.2.3
  ./make-OncoSim

Debian Linux examples, no Docker, you should have c++ installed to run it:

ulimit -s -S 65536
export OM_ROOT=~/openmpp_debian_20250601
export MODEL_DIR=my/OncoSim
~/ompp-docker/ompp-build-linux/make-OncoSim

ulimit -s -S 65536
export MODEL_GIT_URL=https://github.com/StatCan/OncoSim.git
export MODEL_GIT_TAG=v1.2.3.4
export OM_ROOT=~/openmpp_debian_20250601
export ONCOSIM_VARIANT=lung
~/ompp-other/model-build/OncoSim/make-OncoSim

cd ~/tmp
ulimit -s -S 65536
export MODEL_DIR=onco
export OM_ROOT=~/openmpp_debian_20250601
export MODEL_INI=my/OncoSim-test.ini
~/ompp-other/model-build/OncoSim/make-OncoSim

Environment variables:

OM_ROOT                   default: ../..
OM_BUILD_CONFIGS=DEBUG    default: RELEASE
OM_MSG_USE=MPI            default: EMPTY
MODEL_DIR            if not empty then model source code directory
MODEL_GIT_URL        if not empty then git URL of model source code
MODEL_GIT_TAG        if not empty then git tag
MODEL_INI            if not empty then run model after build with this model ini file
MODEL_DOC_DISABLE=0  if =0 or =false then make model documentation, default: MODEL_DOC_DISABLE=1

ONCOSIM_VARIANT      if not empty then model variant: breast gmm cervical colorectal lung allcancers

MODEL_NAME by default: OncoSim or OncoSim-$ONCOSIM_VARIANT
MODEL_DIR  by default: $OM_ROOT/models/OncoSim

if MODEL_DIR not exists and MODEL_GIT_URL specified then do git clone model source code
if MODEL_DIR exists then it must contain code/src-OncoSimX_variants.ompp.txt == copy of original code/OncoSimX_variants.ompp

Examples ONCOSIM_VARIANT:

export ONCOSIM_VARIANT=breast
export ONCOSIM_VARIANT=gmm
export ONCOSIM_VARIANT=cervical
export ONCOSIM_VARIANT=colorectal
export ONCOSIM_VARIANT=lung
export ONCOSIM_VARIANT=allcancers

If ONCOSIM_VARIANT undefined then build "full" OncoSim:

unset ONCOSIM_VARIANT
