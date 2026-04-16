## Using openmpp-build image to build and run OncoSim model variants

To build and (optionally) OncoSim model do:

docker run .... openmpp/openmpp-build:windows-ltsc2025 make-OncoSim

Windows Docker examples:

docker run ^
  -v C:\my\build:C:\build ^
  -e OM_ROOT=ompp ^
  -e MODEL_DIR=ompp\models\OncoSim ^
  openmpp/openmpp-build:windows-ltsc2025 ^
  make-OncoSim

docker run ^
  -v C:\my\build:C:\build ^
  -e OM_ROOT=openmpp_win_20250601 ^
  -e ONCOSIM_VARIANT=lung ^
  -e MODEL_INI=my\test.ini ^
  openmpp/openmpp-build:windows-ltsc2025 ^
  make-OncoSim

docker run ^
  -v C:\my\build:C:\build ^
  -e OM_ROOT=ompp ^
  -e ONCOSIM_VARIANT=lung ^
  -e MODEL_DIR=my\OncoSim ^
  -e MODEL_GIT_URL=https://gitlab.com/my/OncoSim.git
  -e MODEL_GIT_TAG=v1.2.3
  openmpp/openmpp-build:windows-ltsc2025 ^
  make-OncoSim

Windows examples, no Docker, use Visual Studio Command Prompt to run it:

cd \tmp
set OM_ROOT=openmpp_win_20250601
set MODEL_DIR=my\OncoSim
\ompp-other\model-build\OncoSim\make-OncoSim.bat

cd \tmp
set MODEL_GIT_URL=https://gitlab.com/my/OncoSim.git
set OM_ROOT=\my\openmpp
\ompp-other\model-build\OncoSim\make-OncoSim.bat

cd \tmp
set OM_ROOT=\my\openmpp
set ONCOSIM_VARIANT=lung
set MODEL_DIR=my\OncoSim
set MODEL_INI=my\test.ini
set MODEL_GIT_URL=https://gitlab.com/my/OncoSim.git
set MODEL_GIT_TAG=v1.2.3
\my\openmpp\ompp-other\model-build\OncoSim\make-OncoSim.bat

Environment variables:

set OM_ROOT                   default: ..\..
set OM_BUILD_CONFIGS=Debug    default: Release
set OM_BUILD_PLATFORMS=Win32  default: x64
set OM_MSG_USE=MPI            default: EMPTY
set MODEL_DIR            if not empty then source code model directory
set MODEL_GIT_URL        if not empty then git URL of model source code
set MODEL_GIT_TAG        if not empty then git tag
set MODEL_INI            if not empty then run model after build with this model ini file
set MODEL_DOC_DISABLE=0  if =0 or =false then make model documentation, default: do not make model documemtation

set ONCOSIM_VARIANT      if not empty then model variant: breast gmm cervical colorectal lung allcancers
set ONCOSIM_SLN          solution name, default: OncoSimX-ompp.sln

MODEL_NAME by default: OncoSim or OncoSim-%ONCOSIM_VARIANT%
MODEL_DIR  by default: OM_ROOT\models\OncoSim

if MODEL_DIR not exists and MODEL_GIT_URL specified then do git clone model source code
if MODEL_DIR exists then it must contain code\src-OncoSimX_variants.ompp.txt == copy of original code\OncoSimX_variants.ompp

Examples ONCOSIM_VARIANT:

set ONCOSIM_VARIANT=breast
set ONCOSIM_VARIANT=gmm
set ONCOSIM_VARIANT=cervical
set ONCOSIM_VARIANT=colorectal
set ONCOSIM_VARIANT=lung
set ONCOSIM_VARIANT=allcancers

If ONCOSIM_VARIANT undefined then build "full" OncoSim:

set ONCOSIM_VARIANT=
