# Docker image to run stcopenmpp models on Debian stable
#
# Examples of build and arguments default values:
#   docker build -t stcopenmpp/stcopenmpp-run:debian .
#   docker build -t stcopenmpp/stcopenmpp-run:debian --build-arg OMPP_USER=ompp .
#   docker build -t stcopenmpp/stcopenmpp-run:debian --build-arg OMPP_GROUP=ompp .
#   docker build -t stcopenmpp/stcopenmpp-run:debian --build-arg OMPP_UID=1999 .
#   docker build -t stcopenmpp/stcopenmpp-run:debian --build-arg OMPP_GID=1999 .
#
# Examples of run:
#   docker run \
#     -v $HOME/models:/home/models \
#     -e OMPP_USER=models -e OMPP_GROUP=models -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
#     stcopenmpp/stcopenmpp-run:debian \
#     ./MyModel
#
#   docker run \
#     -v $HOME/models:/home/models \
#     -e OMPP_USER=models -e OMPP_GROUP=models -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
#     stcopenmpp/stcopenmpp-run:debian \
#     mpiexec -n 2 MyModel_mpi -OpenM.SubValues 16
#
#   docker run -it stcopenmpp/stcopenmpp-run:debian bash
#
#   docker run \
#     -v $HOME:/home/${USER} \
#     -e OMPP_USER=${USER} -e OMPP_GROUP=`id -gn` -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
#     -it stcopenmpp/stcopenmpp-run:debian \
#     bash

FROM debian:stable

# disable debconf terminal input
ARG DEBIAN_FRONTEND=noninteractive

# update base image
RUN apt-get update
RUN apt-get install -y apt-utils
RUN apt-get upgrade -y


# install Open MPI and SQLite
RUN apt-get install -y sqlite3 && \
  apt-get install -y openmpi-bin && \
  apt-get autoclean

# download and install unixODBC (optional)
RUN apt-get install -y unixodbc && \
  apt-get autoclean

# set local openM++ timezone
RUN rm -f /etc/localtime && \
  ln -s /usr/share/zoneinfo/America/Toronto /etc/localtime

# copy entry point and build scripts
COPY entrypoint.sh /scripts/

RUN chmod 755 /scripts/entrypoint.sh

# Done with installation
# set user, work directory and entry point
#
ARG OMPP_USER=ompp
ARG OMPP_GROUP=ompp
ARG OMPP_UID=1999
ARG OMPP_GID=1999

ENV OMPP_USER  ${OMPP_USER}
ENV OMPP_GROUP ${OMPP_GROUP}
ENV OMPP_UID   ${OMPP_UID}
ENV OMPP_GID   ${OMPP_GID}

# home directory and user are set by entrypoint.sh
#
# RUN groupadd -g $OMPP_GID ${OMPP_GROUP} && \
#   useradd --no-log-init -g ${OMPP_GROUP} -u $OMPP_UID ${OMPP_USER}
# 
# USER ${OMPP_USER}:${OMPP_GROUP}
# 
# WORKDIR /home/${OMPP_USER}/models

ENTRYPOINT ["/scripts/entrypoint.sh"]

# CMD ["cat", "/scripts/README.txt"]

