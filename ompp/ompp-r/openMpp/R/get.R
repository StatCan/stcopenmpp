#' @title Return list of languages
#' @description Returns the list of languages from the `lang_lst` table in the database.
#' @export
#' @param dbCon Database connection.
#'
#' @details
#' Reads the language list from the database and returns it as a data frame.
#'
#' @return A data frame with columns:
#' \describe{
#'   \item{lang_id}{Language ID.}
#'   \item{lang_code}{Language code.}
#'   \item{lang_name}{Language name.}
#' }
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' langRs <- getLanguages(theDb)
#'
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

getLanguages <- function(dbCon)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  # get lang_lst rows: at least one row expected
  langRs <- dbGetQuery(
    dbCon,
    "SELECT lang_id, lang_code, lang_name FROM lang_lst ORDER BY 1"
  )
  if (nrow(langRs) <= 0) stop("invalid database: no language(s) found")

  return(langRs)
}

#' @title Return working set ID by name
#' @description Returns the ID of a working set given its name.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param worksetName Name of the parameters working set.
#'
#' @details
#' Returns the working set ID for the specified name.
#' If the model has no working set with that name, a negative value is returned.
#' If the model has multiple working sets with the same name, the minimum set ID is returned.
#'
#' @return
#' Positive integer on success (working set ID), or negative if not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getDefaultWorksetId}}, \code{\link{getWorksetRunIds}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model by name (if only one version exists)
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get default working set ID by name (expected to exist)
#' setId <- getWorksetIdByName(theDb, defRs, defRs$modelDic$model_name)
#'
#' # try to find "myData" working set ID
#' setId <- getWorksetIdByName(theDb, defRs, "myData")
#' if (setId <= 0L) warning("workset not found: ", "myData")
#'
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

getWorksetIdByName <- function(dbCon, defRs, worksetName)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  if (missing(worksetName)) stop("invalid (missing) workset name")
  if (is.null(worksetName) || is.na(worksetName) || !is.character(worksetName)) stop("invalid or empty workset name")

  # get first set id with specified name for that model id
  setRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MIN(set_id) FROM workset_lst",
      " WHERE model_id = ", defRs$modelDic$model_id,
      " AND set_name = ", toQuoted(worksetName),
      sep=""
    )
  )
  if (nrow(setRs) <= 0) return(-1L)

  return(ifelse(!is.na(setRs[1,1]), as.integer(setRs[1,1]), -1L))
}

#' @title Return default working set ID
#' @description Returns the default working set ID for the model.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#'
#' @details
#' Returns the ID of the model's "default" working set of parameters.
#' Each model must have a default workset. The default workset is the first workset
#' of the model, i.e., the one with the minimum `set_id`.
#' Execution stops if no default workset is found.
#'
#' @return
#' Positive integer (working set ID) on success. Stops execution if default workset is not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getFirstRunId}}, \code{\link{getLastRunId}},
#' \code{\link{getWorksetIdByName}}, \code{\link{getWorksetRunIds}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model by name (if only one version exists)
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get default working set ID (expected to exist)
#' setId <- getDefaultWorksetId(theDb, defRs)
#'
#' # try to find "myData" working set ID
#' setId <- getWorksetIdByName(theDb, defRs, "myData")
#' if (setId <= 0L) warning("workset not found: ", "myData")
#'
#' dbDisconnect(theDb)

getDefaultWorksetId <- function(dbCon, defRs)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  # get first set id for that model id
  setRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MIN(set_id) FROM workset_lst WHERE model_id = ", defRs$modelDic$model_id,
      sep=""
    )
  )
  # one row expected else model id is invalid
  if (is.null(setRs) || nrow(setRs) != 1) {
    stop("no worksets not found for model: ", i_defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
  }

  return(ifelse(!is.na(setRs[1,1]), as.integer(setRs[1,1]), -1L))
}

#' @title Return IDs of model run results for a specified working set
#' @description Returns IDs of model run results where input parameters are from a specified working set.
#' @export
#' @param dbCon Database connection.
#' @param worksetId ID of parameters working set (must be a positive integer).
#'
#' @details
#' Returns IDs of model run results where input parameters are from the specified working set.
#'
#' Note: There is no established link in the database between an input data working set and model run results.
#' Input data can be modified or deleted after a model run. If you need the input parameter values for a specific run,
#' use \code{\link{selectRunParameter}} instead.
#'
#' It is recommended to create a modeling task using \code{\link{createTask}}
#' to include multiple working sets of input parameters and run the model using these inputs.
#'
#' @return Data frame with an integer column \code{$run_id}.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{createTask}}, \code{\link{getModel}}, \code{\link{getFirstRunId}},
#' \code{\link{getLastRunId}}, \code{\link{getDefaultWorksetId}},
#' \code{\link{getWorksetIdByName}}, \code{\link{selectRunParameter}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' runIdRs <- getWorksetRunIds(theDb, 2L)
#'
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

getWorksetRunIds <- function(dbCon, worksetId)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(worksetId)) stop("invalid (missing) workset id")
  if (is.null(worksetId) || is.na(worksetId) || !is.integer(worksetId)) stop("invalid or empty workset id")
  if (worksetId <= 0L) stop("workset id must be positive: ", worksetId)

  # get run ids by predefined option key: OpenM.SetId
  runRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT RL.run_id",
      " FROM run_lst RL",
      " INNER JOIN run_option RO ON (RO.run_id = RL.run_id)",
      " WHERE RL.status = 's'",
      " AND RO.option_key = 'OpenM.SetId'",
      " AND RO.option_value = ", toQuoted(worksetId),
      " ORDER BY 1",
      sep=""
    )
  )
  # it can be empty result with nrow() = 0

  return(runRs)
}

#' @title Return ID of first model run results
#' @description Returns the ID of the first model run results for a given model.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#'
#' @details
#' Returns the ID of the first model run results. This is a positive integer.
#' If the model does not have any run results (the model was never executed), the function returns a negative value.
#'
#' @return Run ID: positive integer on success or negative if not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getLastRunId}}, \code{\link{getModel}}, \code{\link{getTaskIdByName}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model by name (use this if only one version of the model exists)
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get first run ID of the model
#' runId <- getFirstRunId(theDb, defRs)
#' if (runId <= 0L) warning("model run results not found")
#'
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

getFirstRunId <- function(dbCon, defRs)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  # get first run id for that model id
  runRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MIN(run_id) FROM run_lst WHERE model_id = ", defRs$modelDic$model_id,
      sep=""
    )
  )
  # one row expected else model id is invalid
  if (is.null(runRs) || nrow(runRs) != 1) return -1L

  return(ifelse(!is.na(runRs[1,1]), as.integer(runRs[1,1]), -1L))
}

#' @title Return ID of last model run results
#' @description Returns the ID of the last (most recent) model run results for a given model.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#'
#' @details
#' Returns the ID of the last model run results. This is a positive integer.
#' If the model does not have any run results (the model was never executed), the function returns a negative value.
#'
#' @return Run ID: positive integer on success or negative if not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getFirstRunId}}, \code{\link{getModel}}, \code{\link{getWorksetRunIds}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model by name (use this if only one version of the model exists)
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get last run ID of the model
#' runId <- getLastRunId(theDb, defRs)
#' if (runId <= 0L) warning("model run results not found")
#'
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

getLastRunId <- function(dbCon, defRs)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  # get first run id for that model id
  runRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MAX(run_id) FROM run_lst WHERE model_id = ", defRs$modelDic$model_id,
      sep=""
    )
  )
  # one row expected else model id is invalid
  if (is.null(runRs) || nrow(runRs) != 1) return -1L

  return(ifelse(!is.na(runRs[1,1]), as.integer(runRs[1,1]), -1L))
}

#' @title Return modeling task ID by name
#' @description Returns the ID of a modeling task given its name.
#' @export
#' @param dbCon Database connection.
#' @param defRs Optional model definition: database rows describing model input parameters and output tables. Default is NA.
#' @param taskName Name of the modeling task.
#'
#' @details
#' Returns the ID of the modeling task matching the specified name.
#' If no task exists with that name, a negative value is returned.
#' If multiple tasks have the same name, the minimum task ID is returned.
#'
#' If the \code{defRs} argument is supplied, the search is restricted to the specific model.
#'
#' A modeling task is a named set of model inputs that contains a name and a vector of model workset IDs.
#' See \code{\link{createWorkset}} for details about worksets (working sets of model input parameters).
#'
#' Modeling tasks are a convenient way to bundle multiple inputs of the model.
#' Once a task is created, the model can be run by specifying the task name or task ID.
#' The model will iterate through the task's input worksets and produce output results for each input.
#'
#' @return Modeling task ID: positive integer on success or negative if not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{createTask}}, \code{\link{selectTask}}, \code{\link{selectTaskRun}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # try to find "taskOne" task ID for any model
#' taskId <- getTaskIdByName(theDb, taskName = "taskOne")
#' if (taskId <= 0L) warning("task not found: ", "taskOne")
#'
#' # get model by name: use this if only one version of the model exists
#' defRs <- getModel(theDb, "modelOne")
#'
#' # try to find "taskOne" task ID for "modelOne"
#' taskId <- getTaskIdByName(theDb, defRs, "taskOne")
#' if (taskId <= 0L) warning("task not found: ", "taskOne")
#'
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

getTaskIdByName <- function(dbCon, defRs = NA, taskName)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (!missing(defRs) && !is.null(defRs) && !is.na(defRs)) {
    if (!is.list(defRs)) stop("invalid or empty model definition")
  }

  if (missing(taskName)) stop("invalid (missing) task name")
  if (is.null(taskName) || is.na(taskName) || !is.character(taskName)) stop("invalid or empty task name")

  # get first task id with specified name for that model id
  sql <- ifelse(
    !missing(defRs) && !is.null(defRs) && !is.na(defRs),
    paste(
      "SELECT MIN(task_id) FROM task_lst",
      " WHERE model_id = ", defRs$modelDic$model_id,
      " AND task_name = ", toQuoted(taskName),
      sep=""
    ),
    paste(
      "SELECT MIN(task_id) FROM task_lst WHERE task_name = ", toQuoted(taskName),
      sep=""
    )
  )
  taskRs <- dbGetQuery(dbCon, sql)

  if (nrow(taskRs) <= 0) return(-1L)

  return(ifelse(!is.na(taskRs[1,1]), as.integer(taskRs[1,1]), -1L))
}

#' @title Return first ID of modeling task run
#' @description Returns the ID of the first run for a specified modeling task.
#' @export
#' @param dbCon Database connection.
#' @param taskId Modeling task ID.
#'
#' @details
#' Returns the ID of the first run associated with the given modeling task.
#' If there are no runs for the specified task, a negative value is returned.
#'
#' @return Task run ID: positive integer on success or negative if not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getTaskLastRunId}}, \code{\link{getModel}}, \code{\link{getWorksetRunIds}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model by name (if only one version exists)
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get task ID of "taskOne"
#' taskId <- getTaskIdByName(theDb, defRs, "taskOne")
#' if (taskId <= 0L) warning("task not found: taskOne")
#'
#' # get first task run ID
#' firstId <- getTaskFirstRunId(theDb, taskId)
#' if (firstId <= 0L) stop("task run(s) not found")
#'
#' # get last (most recent) task run ID
#' lastId <- getTaskLastRunId(theDb, taskId)
#' if (lastId <= 0L) warning("task run(s) not found")
#'
#' dbDisconnect(theDb)

getTaskFirstRunId <- function(dbCon, taskId)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(taskId)) stop("invalid (missing) modeling task id")
  if (is.null(taskId) || is.na(taskId) || !is.numeric(taskId)) stop("invalid or empty modeling task id")

  # get first task run id for that modeling task
  runRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MIN(task_run_id) FROM task_run_lst WHERE task_id = ", taskId,
      sep=""
    )
  )
  # one row expected else model id is invalid
  if (is.null(runRs) || nrow(runRs) != 1) return -1L

  return(ifelse(!is.na(runRs[1,1]), as.integer(runRs[1,1]), -1L))
}

#' @title Return last ID of modeling task run
#' @description Returns the ID of the last (most recent) run for a specified modeling task.
#' @export
#' @param dbCon Database connection.
#' @param taskId Modeling task ID.
#'
#' @details
#' Returns the ID of the most recent run associated with the given modeling task.
#' If there are no runs for the specified task, a negative value is returned.
#'
#' @return Task run ID: positive integer on success or negative if not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getTaskFirstRunId}}, \code{\link{getModel}}, \code{\link{getTaskIdByName}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model by name (if only one version exists)
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get task ID of "taskOne"
#' taskId <- getTaskIdByName(theDb, defRs, "taskOne")
#' if (taskId <= 0L) warning("task not found: taskOne")
#'
#' # get last task run ID
#' lastId <- getTaskLastRunId(theDb, taskId)
#' if (lastId <= 0L) warning("task run(s) not found")
#'
#' dbDisconnect(theDb)

getTaskLastRunId <- function(dbCon, taskId)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(taskId)) stop("invalid (missing) modeling task id")
  if (is.null(taskId) || is.na(taskId) || !is.numeric(taskId)) stop("invalid or empty modeling task id")

  # get last task run id for that modeling task
  runRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MAX(task_run_id) FROM task_run_lst WHERE task_id = ", taskId,
      sep=""
    )
  )
  # one row expected else model id is invalid
  if (is.null(runRs) || nrow(runRs) != 1) return -1L

  return(ifelse(!is.na(runRs[1,1]), as.integer(runRs[1,1]), -1L))
}
