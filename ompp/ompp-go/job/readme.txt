This directory is an example of models run job directories structure:

job/        : this folder must be shared between all oms instances
    active/   : active model runs state
    history/  : completed (success or fail) model runs
    past/     : (optinal) shadow copy of history folder, invisible to end user
    queue/    : model runs queue
    state/    : servers state and, jobs state and oms instances state
           jobs.queue-#-$INSTANCE-#-paused : if this file exist the instance model runs queue is paused
           jobs.queue.all.paused : if this file exist all model runs queues are paused
    job.ini   : job control settings

To use model run jobs use -oms.JobDir option, for example:

cd openmpp-root-dir
bin/oms -oms.JobDir job
