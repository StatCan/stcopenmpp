#' @title Set or clear read-only status of a model parameters working set
#' @description Mark a model parameters working set as read-only or clear its read-only status.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param isReadonly Logical; if \code{TRUE}, mark the working set as read-only, otherwise clear read-only status.
#' @param worksetId ID of the parameters working set, must be positive integer.
#'
#' @details
#' A workset must be **not read-only** to allow parameter updates using \code{\link{updateWorksetParameter}}.
#' Conversely, a workset must be **read-only** to run the model.
#' Typically, you wrap \code{\link{updateWorksetParameter}} calls with \code{setReadonlyWorkset}
#' or \code{\link{setReadonlyDefaultWorkset}} to temporarily allow updates.
#'
#' A workset can be a full set containing all model parameters or a subset containing only some parameters.
#' Each model must have a "default" workset: the first workset of the model with \code{set_id = min(set_id)}, which always contains all parameters.
#'
#' You must use \code{\link{getModel}} to obtain the model definition \code{defRs}.
#'
#' @return
#' Returns the working set ID if successful, or 0L if the workset was not found.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author
#' amc1999
#'
#' @note
#' To run examples you must have \code{modelOne} database \code{modelOne.sqlite} in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getDefaultWorksetId}}, \code{\link{getWorksetIdByName}},
#' \code{\link{copyWorksetParameterFromRun}}, \code{\link{setReadonlyDefaultWorkset}},
#' \code{\link{updateWorksetParameter}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # reset read-only status of workset
#' setId <- 3L
#' if (setReadonlyWorkset(theDb, defRs, FALSE, setId) <= 0L) {
#'   stop("workset not found: ", setId, " for model: ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
#' }
#'
#' # now you can update model parameters
#' # updateWorksetParameter(theDb, defRs, setId, ageSex)
#'
#' # make workset read-only to run the model
#' setReadonlyWorkset(theDb, defRs, TRUE, setId)
#'
#' dbDisconnect(theDb)

setReadonlyWorkset <- function(dbCon, defRs, isReadonly, worksetId)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  if (missing(worksetId)) stop("invalid (missing) workset id")
  if (is.null(worksetId) || is.na(worksetId) || !is.integer(worksetId)) stop("invalid or empty workset id")
  if (worksetId <= 0L) stop("workset id must be positive: ", worksetId)

  if (missing(isReadonly)) stop("invalid (missing) read-only flag")
  if (is.null(isReadonly) || is.na(isReadonly) || !is.logical(isReadonly)) stop("invalid or empty read-only flag")

  # execute in transaction scope
  isTrxCompleted <- FALSE;
  tryCatch({
    dbBegin(dbCon)

    # set readonly flag by workset id
    rdOnlyVal <- ifelse(isReadonly, 1L, 0L)
    setRs <- lockWorksetUsingReadonly(dbCon, defRs, TRUE, worksetId, FALSE, rdOnlyVal)
    if (is.null(setRs) || setRs$is_readonly != rdOnlyVal) {
      stop("workset not found or has invalid read-only status, set id: ", worksetId)
    }

    isTrxCompleted <- TRUE; # completed OK
  },
  finally = {
    ifelse(isTrxCompleted, dbCommit(dbCon), dbRollback(dbCon))
  })
  return(ifelse(isTrxCompleted, worksetId, 0L))
}

#' @title Set or clear read-only status for default working set of model parameters
#' @description Mark the default working set of model parameters as read-only or clear its read-only status.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param isReadonly Logical; if \code{TRUE}, mark the default working set as read-only, otherwise clear read-only status.
#'
#' @details
#' A workset must be **not read-only** to allow parameter updates using \code{\link{updateWorksetParameter}}.
#' Conversely, a workset must be **read-only** to run the model.
#' Typically, you wrap \code{\link{updateWorksetParameter}} calls with \code{setReadonlyDefaultWorkset}
#' or \code{\link{setReadonlyWorkset}} to temporarily allow updates.
#'
#' A workset can be a full set containing all model parameters or a subset containing only some parameters.
#' Each model must have a "default" workset: the first workset of the model with \code{set_id = min(set_id)}, which always contains all parameters.
#'
#' You must use \code{\link{getModel}} to obtain the model definition \code{defRs}.
#'
#' @return
#' Returns the working set ID of the default workset, or 0L if no worksets exist.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author
#' amc1999
#'
#' @note
#' To run examples you must have \code{modelOne} database \code{modelOne.sqlite} in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getDefaultWorksetId}}, \code{\link{getWorksetIdByName}},
#' \code{\link{copyWorksetParameterFromRun}}, \code{\link{setReadonlyWorkset}},
#' \code{\link{updateWorksetParameter}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # reset read-only status of default workset
#' setId <- setReadonlyDefaultWorkset(theDb, defRs, FALSE)
#' if (setId <= 0L) stop("no worksets exist for model: ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
#'
#' # you can update model parameters now
#' # updateWorksetParameter(theDb, defRs, setId, ageSex, salaryAge)
#'
#' # make default workset read-only to run the model
#' setReadonlyDefaultWorkset(theDb, defRs, TRUE)
#'
#' dbDisconnect(theDb)

setReadonlyDefaultWorkset <- function(dbCon, defRs, isReadonly)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  if (missing(isReadonly)) stop("invalid (missing) read-only flag")
  if (is.null(isReadonly) || is.na(isReadonly) || !is.logical(isReadonly)) stop("invalid or empty read-only flag")

  # execute in transaction scope
  setId <- 0L
  isTrxCompleted <- FALSE;
  tryCatch({
    dbBegin(dbCon)

    # set readonly flag by model name and digest
    rdOnlyVal <- ifelse(isReadonly, 1L, 0L)
    setRs <- lockWorksetUsingReadonly(dbCon, defRs, FALSE, 0, FALSE, rdOnlyVal)
    if (is.null(setRs) || setRs$is_readonly != rdOnlyVal) {
      stop("workset not found or has invalid read-only status, model: ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
    }

    setId <- setRs$set_id
    isTrxCompleted <- TRUE; # completed OK
  },
  finally = {
    ifelse(isTrxCompleted, dbCommit(dbCon), dbRollback(dbCon))
  })
  return(ifelse(isTrxCompleted, setId, 0L))
}

#' @title
#' Set read-only flag for a workset (internal)
#'
#' @description
#' **Internal use only. Must be executed within a database transaction.**
#' Set or update the read-only flag for a workset, either by workset ID or by model ID.
#' Returns the corresponding rows from \code{workset_lst} after the update.
#'
#' @export
#'
#' @param dbCon Database connection.
#' @param i_defRs Model definition database rows.
#' @param i_isSetId Logical; if \code{TRUE}, use \code{i_setId} to identify workset, otherwise use default workset by model ID.
#' @param i_setId Workset ID to update (used if \code{i_isSetId = TRUE}).
#' @param i_isAddValue Logical; if \code{TRUE}, add \code{i_readonlyValue} to the existing \code{is_readonly} value, otherwise set it directly.
#' @param i_readonlyValue Integer value to assign or add to \code{is_readonly}.
#'
#' @details
#' This function updates the \code{is_readonly} flag in the \code{workset_lst} table.
#' Behavior depends on \code{i_isAddValue}:
#' \itemize{
#'   \item If \code{i_isAddValue = TRUE}, the current \code{is_readonly} is incremented by \code{i_readonlyValue}.
#'   \item If \code{i_isAddValue = FALSE}, \code{is_readonly} is set directly to \code{i_readonlyValue}.
#' }
#'
#' The function returns a data frame with the updated rows:
#' \itemize{
#'   \item \code{set_id} — workset ID
#'   \item \code{model_id} — model ID
#'   \item \code{is_readonly} — updated read-only flag
#' }
#'
#' @return
#' Data frame of \code{workset_lst} rows with updated \code{is_readonly} values.
#'
#' @note
#' Must be executed inside a database transaction.
#'
#' @keywords internal

lockWorksetUsingReadonly <- function(dbCon, i_defRs, i_isSetId, i_setId, i_isAddValue, i_readonlyValue)
{
  # set output value
  setRs <- NULL

  # expression to update is_readonly
  rdOnlyExpr <- ifelse(i_isAddValue, paste("is_readonly + ", i_readonlyValue, sep = ""), as.character(i_readonlyValue))

  if (i_isSetId) {    # use workset id

    dbExecute(
      dbCon,
      paste(
        "UPDATE workset_lst SET is_readonly = ", rdOnlyExpr,
        " WHERE set_id = ", i_setId,
        " AND model_id = ", i_defRs$modelDic$model_id,
        sep=""
      )
    )
    setRs <- dbGetQuery(
      dbCon,
      paste(
        "SELECT set_id, model_id, is_readonly FROM workset_lst WHERE set_id = ", i_setId,
        sep=""
      )
    )
    # one row expected else set id is invalid
    if (is.null(setRs) || nrow(setRs) != 1 || setRs$model_id != i_defRs$modelDic$model_id) {
      stop("workset not found: ", i_setId, " or not belong to model: ", i_defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
    }
  }
  else {    # use model id

    dbExecute(
      dbCon,
      paste(
        "UPDATE workset_lst SET is_readonly = ", rdOnlyExpr,
        " WHERE set_id = ",
        " (",
        " SELECT MIN(M.set_id) FROM workset_lst M WHERE M.model_id = ", i_defRs$modelDic$model_id,
        " )",
        sep=""
      )
    )
    setRs <- dbGetQuery(
      dbCon,
      paste(
        "SELECT WS.set_id, WS.model_id, WS.is_readonly FROM workset_lst WS WHERE WS.set_id = ",
        " (",
        " SELECT MIN(M.set_id) FROM workset_lst M WHERE M.model_id = ", i_defRs$modelDic$model_id,
        " )",
        sep=""
      )
    )
    # one row expected else model id is invalid
    if (is.null(setRs) || nrow(setRs) != 1) {
      stop("no worksets not found for model: ", i_defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
    }
  }

  return(setRs)
}
