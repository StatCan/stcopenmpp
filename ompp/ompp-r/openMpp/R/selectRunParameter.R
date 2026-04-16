#' @title Select parameter values from model run result
#' @description Retrieve input parameter values from a specific model run result.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param runId Positive integer. ID of model run results.
#' @param paramName Parameter name (character), e.g., `"sexAge"`.
#'
#' @details
#' Returns values of an input parameter from the model run results with the specified `runId`.
#' The `defRs` object must be obtained via `\code{\link{getModel}}`.
#'
#' @return
#' Data frame of database rows with parameter dimensions and values:
#' \describe{
#'   \item{sub_id}{Parameter sub-value ID or zero if parameter has no sub-values.}
#'   \item{dim0,...,dimN}{Dimension items enum IDs (not returned if rank is zero).}
#'   \item{value}{Parameter value.}
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
#' \code{\link{getModel}}, \code{\link{getFirstRunId}}, \code{\link{getLastRunId}},
#' \code{\link{selectRunAccumulator}}, \code{\link{selectRunOutputValue}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # get first run id
#' runId <- getFirstRunId(theDb, defRs)
#' if (runId <= 0L) stop("model run results not found")
#'
#' # select parameter "ageSex" value
#' paramValueRs <- selectRunParameter(theDb, defRs, runId, "ageSex")
#'
#' dbDisconnect(theDb)

selectRunParameter <-  function(dbCon, defRs, runId, paramName)
  {
    # validate input parameters
    if (missing(dbCon)) stop("invalid (missing) database connection")
    if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

    if (missing(defRs)) stop("invalid (missing) model definition")
    if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

    if (missing(runId)) stop("invalid (missing) run id")
    if (is.null(runId) || is.na(runId) || !is.integer(runId)) stop("invalid or empty run id")
    if (runId <= 0L) stop("run id must be positive: ", runId)

    if (missing(paramName)) stop("invalid (missing) parameter name")
    if (is.null(paramName)) stop("invalid or empty parameter name")
    if (is.na(paramName) || !is.character(paramName) || length(paramName) <= 0) {
      stop("invalid or empty parameter name")
    }

    # get parameter row by name
    paramRow <- defRs$paramDic[which(defRs$paramDic$parameter_name == paramName), ]
    if (nrow(paramRow) != 1) {
      stop("parameter not found in model parameters list: ", paramName)
    }

    nRank <- paramRow$parameter_rank
    dbTableName <- paramRow$db_run_table

    # check if run id is belong the model and completed
    rlRow <- dbGetQuery(
      dbCon,
      paste(
        "SELECT run_id FROM run_lst",
        " WHERE run_id = ", runId,
        " AND model_id = ", defRs$modelDic$model_id,
        " AND sub_completed = sub_count",
        sep=""
      )
    )
    if (nrow(rlRow) != 1L) {
      stop("model run results not found (or not completed), run id: ", runId)
    }

    # SELECT sub_id, dim0, dim1, param_value
    # FROM ageSex_p2012_817
    # WHERE run_id = (SELECT base_run_id FROM run_parameter WHERE run_id = 1234 AND parameter_hid = 1)
    # ORDER BY 1, 2, 3
    d <- defRs$paramDims[which(defRs$paramDims$parameter_hid == paramRow$parameter_hid), c("col_name", "dim_name")]

    sqlSel <-
      paste(
        "SELECT sub_id, ",
        ifelse(nRank > 0L,
               paste(
                 paste(d[,1], " AS ", '"', d[,2], '"', sep = "", collapse = ", "),
                 ", ",
                 sep = ""
               ),
               ""
        ),
        " param_value",
        " FROM ", dbTableName,
        " WHERE run_id = ",
        " (SELECT base_run_id FROM run_parameter WHERE run_id = ", runId, " AND parameter_hid = ", paramRow$parameter_hid, ")",
        ifelse(nRank > 0L,
               paste(" ORDER BY ",
                     paste(1L:nRank, sep = "", collapse = ", "),
                     sep = ""
               ),
               ""
        ),
        sep = ""
      )

    selRs <- dbGetQuery(dbCon, sqlSel)
    return(selRs)
  }
