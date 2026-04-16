#' @title Select modeling task run
#' @description Retrieve information about a modeling task run, including status, timing, input worksets, and output results.
#' @export
#' @param dbCon Database connection.
#' @param taskRunId ID of the modeling task run.
#'
#' @details
#' Returns task run status, timestamps, input workset IDs, and output result run IDs.
#'
#' Task run status can be one of:
#' \describe{
#'   \item{i}{Not yet started.}
#'   \item{p}{Run in progress.}
#'   \item{w}{Run in progress, waiting for additional input.}
#'   \item{s}{Completed successfully.}
#'   \item{e}{Failed (error).}
#'   \item{x}{Reserved.}
#' }
#'
#' Status \code{w} indicates that the task can be dynamically updated by an external script.
#' The model executable is waiting for additional input or a "ready to be completed" signal.
#' Use \code{\link{setTaskWaitCompleted}} to mark the task as ready to be completed,
#' and \code{\link{updateTask}} to insert additional input.
#'
#' @return
#' A list of database rows:
#' \describe{
#'   \item{taskRunLst}{task_run_lst row: task run ID, status, and time.}
#'   \item{taskRunSet}{task_run_set rows: input workset IDs and result run IDs.}
#' }
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author
#' amc1999
#'
#' @note
#' To run examples you must have `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getTaskIdByName}}, \code{\link{getTaskFirstRunId}},
#' \code{\link{getTaskLastRunId}}, \code{\link{selectTask}}, \code{\link{selectTaskList}},
#' \code{\link{selectTaskRunList}}, \code{\link{setTaskWaitCompleted}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get task id of "taskOne"
#' taskId <- getTaskIdByName(theDb, defRs, "taskOne")
#' if (taskId <= 0L) stop("task not found: taskOne")
#'
#' # get last (most recent) run of that task
#' lastId <- getTaskLastRunId(theDb, taskId)
#' if (lastId <= 0L) stop("task run(s) not found")
#'
#' # select task run status, input worksets, and output results
#' taskRunRs <- selectTaskRun(theDb, lastId)
#'
#' # select list of all task runs for the task
#' taskRunLstRs <- selectTaskRunList(theDb, taskId)
#'
#' dbDisconnect(theDb)

selectTaskRun <- function(dbCon, taskRunId)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(taskRunId) || is.null(taskRunId) || !is.numeric(taskRunId)) stop("invalid or empty task run id")

  # get task_run_lst row: single row expected
  sql <- paste(
      "SELECT task_run_id, task_id, sub_count, create_dt, status, update_dt FROM task_run_lst",
      " WHERE task_run_id = ", taskRunId,
      " ORDER BY 1",
      sep=""
    )
  taskRunRs <- list(taskRunLst = dbGetQuery(dbCon, sql))

  if (nrow(taskRunRs$taskRunLst) != 1) stop("task run not found, id: ", taskRunId)

  # task_run_set: task input and output
  taskRunRs[["taskRunSet"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT task_run_id, run_id, set_id, task_id FROM task_run_set WHERE task_run_id = ", taskRunId, " ORDER BY 1, 2",
      sep = ""
    )
  )
  # task_run_set may not exist

  return(taskRunRs)
}

#' @title Select list of modeling task runs
#' @description Retrieve the list of runs for a specific modeling task, including task run ID, status, and time.
#' @export
#' @param dbCon Database connection.
#' @param taskId ID of the modeling task.
#'
#' @details
#' Returns for the specified task ID a list of task run IDs, their status, and timestamps.
#'
#' Task run status can be one of:
#' \describe{
#'   \item{i}{Not yet started.}
#'   \item{p}{Run in progress.}
#'   \item{w}{Run in progress, waiting for additional input.}
#'   \item{s}{Completed successfully.}
#'   \item{e}{Failed (error).}
#'   \item{x}{Reserved.}
#' }
#'
#' Status \code{w} indicates that the task can be dynamically updated by an external script.
#' The model executable is waiting for additional input or a "ready to be completed" signal.
#' Use \code{\link{setTaskWaitCompleted}} to mark the task as ready to be completed,
#' and \code{\link{updateTask}} to insert additional input.
#'
#' @return
#' A list of database rows:
#' \describe{
#'   \item{taskLst}{task_lst row: task ID and name.}
#'   \item{taskRunLst}{task_run_lst rows: task run ID, status, and time.}
#' }
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author
#' amc1999
#'
#' @note
#' To run examples you must have `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getTaskIdByName}}, \code{\link{getTaskFirstRunId}},
#' \code{\link{getTaskLastRunId}}, \code{\link{selectTask}}, \code{\link{selectTaskList}},
#' \code{\link{selectTaskRun}}, \code{\link{setTaskWaitCompleted}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get task id of "taskOne"
#' taskId <- getTaskIdByName(theDb, defRs, "taskOne")
#' if (taskId <= 0L) stop("task not found: taskOne")
#'
#' # last (most recent) run of that task
#' lastId <- getTaskLastRunId(theDb, taskId)
#' if (lastId <= 0L) stop("task run(s) not found")
#'
#' # select task run status, input worksets, and output results
#' taskRunRs <- selectTaskRun(theDb, lastId)
#'
#' # select list of all task runs for the task
#' taskRunLstRs <- selectTaskRunList(theDb, taskId)
#'
#' dbDisconnect(theDb)

selectTaskRunList <- function(dbCon, taskId)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(taskId) || is.null(taskId) || !is.numeric(taskId)) stop("invalid or empty modeling task id")

  # get task_lst row: single row expected
  sql <- paste(
      "SELECT task_id, model_id, task_name FROM task_lst",
      " WHERE task_id = ", taskId,
      " ORDER BY 1",
      sep=""
    )
  taskRunLstRs <- list(taskLst = dbGetQuery(dbCon, sql))

  if (nrow(taskRunLstRs$taskLst) != 1) stop("modeling task not found, id: ", taskId)

  # task_run_lst rows:
  taskRunLstRs[["taskRunLst"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT task_run_id, task_id, sub_count, create_dt, status, update_dt FROM task_run_lst",
      " WHERE task_id = ", taskId,
      " ORDER BY 1",
      sep = ""
    )
  )

  return(taskRunLstRs)
}
