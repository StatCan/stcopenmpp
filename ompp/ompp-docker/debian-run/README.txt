## Using image

To run stcopenmpp model do:

  docker run .... stcopenmpp/stcopenmpp-run:debian ./MyModel
  
Examples:
  docker run \
    -v $HOME/models:/home/models \
    -e OMPP_USER=models -e OMPP_GROUP=models -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
    stcopenmpp/stcopenmpp-run:debian \
    ./MyModel

  docker run \
    -v $HOME/models:/home/models \
    -e OMPP_USER=models -e OMPP_GROUP=models -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
    stcopenmpp/stcopenmpp-run:debian \
    mpiexec -n 2 MyModel_mpi -OpenM.SubValues 16

To start shell do:

  docker run -it stcopenmpp/stcopenmpp-run:debian bash

  docker run \
    -v $HOME:/home/${USER} \
    -e OMPP_USER=${USER} -e OMPP_GROUP=`id -gn` -e OMPP_UID=$UID -e OMPP_GID=`id -g` \
    -it stcopenmpp/stcopenmpp-run:debian bash

Environment variables:
  OMPP_USER=ompp   # default: ompp, container user name and HOME
  OMPP_GROUP=ompp  # default: ompp, container group name
  OMPP_UID=1999    # default: 1999, container user ID
  OMPP_GID=1999    # default: 1999, container group ID
