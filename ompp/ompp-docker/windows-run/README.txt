## Using image

To run stcopenmpp model do:

  docker run .... stcopenmpp/stcopenmpp-run:windows MyModel.exe

  Examples:
  docker run -v C:\my\models\bin:C:\ompp stcopenmpp/stcopenmpp-run:windows MyModel.exe
  docker run -v C:\my\models\bin:C:\ompp stcopenmpp/stcopenmpp-run:windows mpiexec -n 2 MyModel_mpi.exe -OpenM.SubValues 16
  docker run -v C:\my\models\bin:C:\ompp -e OM_ROOT=C:\ompp stcopenmpp/stcopenmpp-run:windows MyModel.exe
  
To start command prompt do:
  docker run -v C:\my\models\bin:C:\ompp -it stcopenmpp/stcopenmpp-run:windows
