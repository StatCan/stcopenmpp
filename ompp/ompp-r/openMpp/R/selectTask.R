#' @title Select modeling task
#' @description Retrieve modeling task text (name, description, notes) and input workset IDs.
#' @export
#' @param dbCon Database connection.
#' @param taskId Modeling task ID (positive integer).
#'
#' @details
#' Returns the text and input worksets of a modeling task:
#' - Task text includes the task name, description, and optional notes in multiple languages.
#' - Task input worksets are the list of working set IDs associated with the task.
#'
#' @return
#' A list of database rows:
#' \describe{
#'   \item{taskLst}{task_lst row: task ID and name.}
#'   \item{taskTxt}{task_txt rows: language, description, notes.}
#'   \item{taskSet}{task_set rows: task input workset IDs.}
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
#' \code{\link{getTaskLastRunId}}, \code{\link{selectTaskList}},
#' \code{\link{selectTaskRun}}, \code{\link{selectTaskRunList}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # find task ID
#' taskId <- getTaskIdByName(theDb, defRs, "taskOne")
#' if (taskId <= 0L) stop("task: ", "taskOne", " not found for model: ",
#'                        defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
#'
#' # select task text and input worksets
#' taskRs <- selectTask(theDb, taskId)
#'
#' # select list of all tasks for the model
#' taskLstRs <- selectTaskList(theDb, defRs)
#'
#' dbDisconnect(theDb)

selectTask <- function(dbCon, taskId)
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
  taskRs <- list(taskLst = dbGetQuery(dbCon, sql))

  if (nrow(taskRs$taskLst) != 1) stop("modeling task not found, id: ", taskId)

  # task text: language, description, notes
  taskRs[["taskTxt"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT T.task_id, T.lang_id, L.lang_code, T.descr, T.note",
      " FROM task_txt T INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)",
      " WHERE T.task_id = ", taskId,
      " ORDER BY 1, 2",
      sep = ""
    )
  )
  # task_txt may not exist

  # task sets: workset id's
  taskRs[["taskSet"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT task_id, set_id FROM task_set WHERE task_id = ", taskId, " ORDER BY 1, 2",
      sep = ""
    )
  )
  # task_set may not exist

  return(taskRs)
}

#' @title Select list of modeling tasks
#' @description Retrieve the list of modeling tasks for a specified model.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#'
#' @details
#' Returns the list of modeling tasks for the specified model. Each task includes its text (name, description, notes)
#' and ID. Useful for enumerating all tasks available for a given model.
#'
#' @return
#' A list of database rows:
#' \describe{
#'   \item{taskLst}{task_lst rows: task ID and name.}
#'   \item{taskTxt}{task_txt rows: language, description, notes.}
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
#' \code{\link{getTaskLastRunId}}, \code{\link{selectTask}}, \code{\link{selectTaskRun}},
#' \code{\link{selectTaskRunList}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # select list of all tasks for the model
#' taskLstRs <- selectTaskList(theDb, defRs)
#'
#' dbDisconnect(theDb)

selectTaskList <- function(dbCon, defRs)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  # get task_lst rows
  sql <- paste(
      "SELECT task_id, model_id, task_name FROM task_lst",
      " WHERE model_id = ", defRs$modelDic$model_id,
      " ORDER BY 1",
      sep=""
    )
  taskLstRs <- list(taskLst = dbGetQuery(dbCon, sql))

  # task text: language, description, notes
  taskLstRs[["taskTxt"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT T.task_id, T.lang_id, L.lang_code, T.descr, T.note",
      " FROM task_txt T",
      " INNER JOIN task_lst M ON (M.task_id = T.task_id)",
      " INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)",
      " WHERE M.model_id = ", defRs$modelDic$model_id,
      " ORDER BY 1, 2",
      sep = ""
    )
  )

  return(taskLstRs)
}
