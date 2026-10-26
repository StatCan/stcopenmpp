# Linux / macOS build of BMIGen with OpenM++.
# Set OM_ROOT to your OpenM++ installation, e.g.  export OM_ROOT=$HOME/openmpp
# Then:  make RELEASE=1 && make RELEASE=1 publish
MODEL_NAME = BMIGen
ifndef OM_ROOT
  OM_ROOT = ../..
endif

export OM_ROOT

# OpenM++ framework sources are Windows-1252; model sources are UTF-8 with BOM.
ifndef OMC_CODE_PAGE
  OMC_CODE_PAGE = WINDOWS-1252
endif

include $(OM_ROOT)/models/makefile.model.mak
