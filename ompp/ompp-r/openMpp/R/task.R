#' @title Create new modeling task
#' @description Create a new modeling task, which is a named set of multiple model inputs.
#' @export
#' @usage
#' createTask(dbCon, defRs, taskTxt = NA, setIds = NA)
#'
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param taskTxt Optional. Data frame specifying modeling task text:
#' \itemize{
#'   \item \code{$name} — modeling task name (must be unique)
#'   \item \code{$lang} — language code
#'   \item \code{$descr} — modeling task description
#'   \item \code{$note} — optional task notes
#' }
#' @param setIds Optional. Modeling task inputs: vector of integer workset IDs, or a data frame with a \code{$set_id} column.
#'        Can supply a single workset ID or multiple IDs.
#'
#' @details
#' This function creates a new modeling task with optional text data (name, description, or notes)
#' and optional input working sets. You can create an empty task without any inputs or text and later update it
#' using \code{\link{updateTask}}.
#'
#' Modeling task is a convenient way to bundle together multiple model inputs. After creation,
#' you can run the model by specifying the task name or task ID. The model will iterate through the task's
#' input worksets and produce output results for each.
#'
#' @return
#' Returns the ID of the new modeling task or 0L on error.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author
#' amc1999
#'
#' @note
#' To run examples you must have the modelOne database (\code{modelOne.sqlite}) in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getTaskIdByName}}, \code{\link{updateTask}}
#'
#' @examples
#' myTaskTxt <- data.frame(
#'   name = "myTask",
#'   lang = c("EN", "FR"),
#'   descr = c("my first modeling task", "description in French"),
#'   note = c("this is a test task and includes two model input data sets with id 2 and 4", NA),
#'   stringsAsFactors = FALSE
#' )
#'
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' defRs <- getModel(theDb, "modelOne")
#'
#' taskId <- createTask(theDb, defRs, myTaskTxt, c(2, 4))
#' if (taskId <= 0L) stop("task creation failed: ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
#'
#' dbDisconnect(theDb)
#' # You can now run the model with the new task:
#' #   modelOne -OpenM.TaskName myTask
#'
#' @keywords OpenM++ database

createTask <- function(dbCon, defRs, taskTxt = NA, setIds = NA)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs))|| !is.list(defRs)) stop("invalid or empty model definition")

  # create new task in transaction scope
  taskId <- 0L

  isTrxCompleted <- FALSE;
  tryCatch({
    dbBegin(dbCon)

    # get next task id
    dbExecute(dbCon, "UPDATE id_lst SET id_value = id_value + 1 WHERE id_key = 'run_id_set_id'")
    idRs <- dbGetQuery(dbCon, "SELECT id_value FROM id_lst WHERE id_key = 'run_id_set_id'")
    if (nrow(idRs) <= 0L || idRs$id_value <= 0L) stop("can not get new task id from id_lst table")

    taskId <- idRs$id_value

    # create task with auto-name
    dbExecute(
      dbCon,
      paste(
        "INSERT INTO task_lst (task_id, model_id, task_name) VALUES (",
        taskId, ", ",
        defRs$modelDic$model_id, ", ",
        toQuoted(paste("task_", taskId, sep = "")), " )",
        sep = ""
      )
    )

    # set task text: name, description notes
    updateTaskTxt(dbCon, defRs, taskId, taskTxt)

    # append workset ids
    updateTaskSetIds(dbCon, taskId, setIds)

    isTrxCompleted <- TRUE; # completed OK
  },
  finally = {
    ifelse(isTrxCompleted, dbCommit(dbCon), dbRollback(dbCon))
  })
  return(ifelse(isTrxCompleted, taskId, 0L))
}

#' @title Update modeling task
#' @description Update an existing modeling task with new text (name, description, notes) or additional input working sets.
#' @export
#' @usage
#' updateTask(dbCon, defRs, taskId, taskTxt = NA, setIds = NA)
#'
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param taskId Modeling task ID (must already exist).
#' @param taskTxt Optional. Data frame specifying modeling task text updates:
#' \itemize{
#'   \item \code{$name} — optional task name
#'   \item \code{$lang} — language code
#'   \item \code{$descr} — task description
#'   \item \code{$note} — optional task notes
#' }
#' @param setIds Optional. Additional input worksets to include in the task:
#' vector of workset IDs or a data frame with a \code{$set_id} column.
#'
#' @details
#' This function updates an existing modeling task by modifying its text data (name, description, notes)
#' and/or adding input working sets. A modeling task is a named set of model inputs, and each input is
#' represented by a workset (working set of model parameters).
#'
#' After updating, you can run the model using the task name or task ID, and the model will iterate
#' through all input worksets, producing output results for each.
#'
#' @return
#' Returns the task ID on success or 0L on error.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author
#' amc1999
#'
#' @note
#' To run examples you must have the modelOne database (\code{modelOne.sqlite}) in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getTaskIdByName}}, \code{\link{createTask}}
#'
#' @examples
#' myTaskTxt <- data.frame(
#'   name = "myTask",
#'   lang = c("EN", "FR"),
#'   descr = c("my first modeling task", "description in French"),
#'   note = c("this is a test task and includes two model input data sets with id 2 and 4", NA),
#'   stringsAsFactors = FALSE
#' )
#'
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' defRs <- getModel(theDb, "modelOne")
#'
#' # create new task, initially empty
#' taskId <- createTask(theDb, defRs)
#' if (taskId <= 0L) stop("task creation failed: ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
#'
#' # update task with new text and two input data sets
#' taskId <- updateTask(theDb, defRs, taskId, myTaskTxt, c(2, 4))
#' if (taskId <= 0L) stop("task update failed, id: ", taskId, ", ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
#'
#' dbDisconnect(theDb)
#' # You can now run the model with the updated task:
#' #   modelOne -OpenM.TaskName myTask
#'
#' @keywords OpenM++ database

updateTask <- function(dbCon, defRs, taskId, taskTxt = NA, setIds = NA)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  if (missing(taskId) || is.null(taskId) || !is.numeric(taskId)) stop("invalid or empty modeling task id")

  # update existing task in transaction scope
  isTrxCompleted <- FALSE;

  tryCatch({
    dbBegin(dbCon)

    # check if task exist
    idRs <- dbGetQuery(dbCon,
                       paste(
                         "SELECT task_id FROM task_lst WHERE task_id = ", taskId,
                         sep = ""
                       )
    )
    if (nrow(idRs) <= 0L || idRs$task_id != taskId) stop("invalid (non-existing) task id :", taskId)

    # set task text: name, description notes
    updateTaskTxt(dbCon, defRs, taskId, taskTxt)

    # append workset ids
    updateTaskSetIds(dbCon, taskId, setIds)

    isTrxCompleted <- TRUE; # completed OK
  },
  finally = {
    ifelse(isTrxCompleted, dbCommit(dbCon), dbRollback(dbCon))
  })
  return(ifelse(isTrxCompleted, taskId, 0L))
}

#' @title Set modeling task "wait completed" status
#' @description Signal a currently running modeling task that it is "ready to be completed".
#' @export
#' @usage
#' setTaskWaitCompleted(dbCon, taskRunId, isWaitCompleted = FALSE)
#'
#' @param dbCon Database connection.
#' @param taskRunId ID of the modeling task run.
#' @param isWaitCompleted Logical. If TRUE, signals the running model that the task is ready to be completed.
#'
#' @details
#' Use this function to mark a running task as "ready to be completed".
#' Task run status can be one of:
#' \itemize{
#'   \item \code{i} — not yet started
#'   \item \code{p} — run in progress
#'   \item \code{w} — run in progress, wait for additional input
#'   \item \code{s} — completed successfully
#'   \item \code{e} — failed (error)
#'   \item \code{x} — reserved
#' }
#'
#' Status \code{w} means the task can be dynamically updated by an external script.
#' The model executable waits for additional input or for the "ready to be completed" signal.
#' Use \code{updateTask} to insert additional task input.
#'
#' The model executable must be running with the \code{-OpenM.TaskWait true} argument to handle dynamic task input.
#' Example:
#' \preformatted{
#' modelOne -OpenM.TaskName taskOne -OpenM.TaskWait true
#' }
#'
#' @return
#' Returns the task ID on success or 0L on error.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author
#' amc1999
#'
#' @note
#' To run examples, the modelOne database (\code{modelOne.sqlite}) must exist in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getTaskIdByName}}, \code{\link{createTask}}, \code{\link{updateTask}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' defRs <- getModel(theDb, "modelOne")
#' taskId <- getTaskIdByName(theDb, defRs, "taskOne")
#' if (taskId <= 0L) stop("task not found")
#'
#' # last (most recent) run of that task
#' lastId <- getTaskLastRunId(theDb, taskId)
#' if (lastId <= 0L) stop("task run(s) not found")
#'
#' # signal that the running task is ready to be completed
#' setTaskWaitCompleted(theDb, lastId, TRUE)
#'
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

setTaskWaitCompleted <- function(dbCon, taskRunId, isWaitCompleted = FALSE)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(taskRunId) || is.null(taskRunId) || !is.numeric(taskRunId)) stop("invalid or empty task run id")

  # update task in transaction scope
  isTrxCompleted <- FALSE;

  tryCatch({
    dbBegin(dbCon)

    # check if task exist
    dbExecute(dbCon,
      paste(
        "UPDATE task_run_lst",
        " SET status = CASE",
        " WHEN status = 'w' THEN ", ifelse(isWaitCompleted, "'p'", "'w'"),
        " ELSE status",
        " END,",
        " update_dt = ", toQuoted(format(Sys.time(), "%Y-%m-%d %H:%M:%S")),
        " WHERE task_run_id = ", taskRunId,
        sep = ""
      )
    )

    isTrxCompleted <- TRUE; # completed OK
  },
  finally = {
    ifelse(isTrxCompleted, dbCommit(dbCon), dbRollback(dbCon))
  })
  return(ifelse(isTrxCompleted, taskRunId, 0L))
}
