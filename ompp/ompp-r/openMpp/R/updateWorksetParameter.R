#' @title Update parameters working set with new values and value notes
#' @description
#' Update parameter values and optional value notes in a specific model workset.
#' @export
#' @usage
#' updateWorksetParameter(dbCon, defRs, worksetId, ...)
#'
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param worksetId ID of parameters working set (must be positive integer).
#' @param ... List of parameter updates. Each element is a list with:
#' \describe{
#'   \item{\$name}{Parameter name (character).}
#'   \item{\$subCount}{(Optional) Number of parameter sub-values; default is 1.}
#'   \item{\$subId}{(Optional) Sub-value ID; default is 0.}
#'   \item{\$value}{Parameter value (scalar, vector, or data frame). If data frame, must have columns: \$dimName0, \$dimName1,..., \$value.}
#'   \item{\$txt}{(Optional) Parameter value notes, data frame with \$lang (language code) and \$note (note text).}
#' }
#'
#' @details
#' This function allows updating parameter values and notes in a specific workset.
#'
#' - Parameters can have multiple sub-values (default is 1, subId=0).
#' - Workset must be not read-only to allow updates.
#' - Workset must be read-only to run the model; use \code{setReadonlyWorkset} or \code{setReadonlyDefaultWorkset} to toggle status.
#' - Default workset (first workset for a model, set_id = min(set_id)) always includes all parameters.
#' - To create a new workset as a full set, supply all parameters via \dots to \code{createWorkset}.
#' - To create a subset of parameters, use \code{createWorksetBasedOnRun} with an existing model run.
#' - Each workset has a unique ID and name; find ID by name using \code{getWorksetIdByName}.
#' - Must use \code{getModel} to obtain the model definition (\code{defRs}).
#'
#' @return ID of the workset or 0L on error.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' Requires a local database file (e.g., modelOne.sqlite) in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getDefaultWorksetId}}, \code{\link{getWorksetIdByName}},
#' \code{\link{createWorkset}}, \code{\link{createWorksetBasedOnRun}}, \code{\link{copyWorksetParameterFromRun}},
#' \code{\link{setReadonlyWorkset}}, \code{\link{setReadonlyDefaultWorkset}}
#'
#' @examples
#' # Define ageSex parameter with sub-value 0
#' ageSex <- list(
#'   name = "ageSex",
#'   subId = 0L,
#'   value = c(10, rep(c(1, 2, 3), times = 2), 20),
#'   txt = data.frame(
#'     lang = c("EN", "FR"),
#'     note = c("age by sex value notes", NA),
#'     stringsAsFactors = FALSE
#'   )
#' )
#'
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400"))
#'
#' defRs <- getModel(theDb, "modelOne")
#' setId <- 4L
#' if (setReadonlyWorkset(theDb, defRs, FALSE, setId) <= 0L) {
#'   stop("workset not found: ", setId)
#' }
#'
#' updateWorksetParameter(theDb, defRs, setId, ageSex)
#' setReadonlyWorkset(theDb, defRs, TRUE, setId)
#'
#' dbDisconnect(theDb)

updateWorksetParameter <- function(dbCon, defRs, worksetId, ...)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  if (missing(worksetId)) stop("invalid (missing) workset id")
  if (is.null(worksetId) || is.na(worksetId) || !is.integer(worksetId)) stop("invalid or empty workset id")
  if (worksetId <= 0L) stop("workset id must be positive: ", worksetId)

  # get list of languages and validate parameters data
  wsParamLst <- list(...)
  if (length(wsParamLst) <= 0) stop("invalid (missing) workset parameters list")

  if (!validateParameterValueLst(defRs$langLst, FALSE, wsParamLst)) return(0L)

  # execute in transaction scope
  isTrxCompleted <- FALSE;
  tryCatch({
    dbBegin(dbCon)

    # find model by workset id, it must not be readonly workset
    setRs <- lockWorksetUsingReadonly(dbCon, defRs, TRUE, worksetId, TRUE, -1L)
    if (setRs$is_readonly != -1L) {
      stop("workset is read-only (or invalid): ", worksetId)
    }

    # check if supplied parameters are in model: parameter_name in parameter_dic table
    # check if supplied parameters are in workset_parameter table
    setParamRs <- dbGetQuery(
      dbCon,
      paste(
        "SELECT set_id, parameter_hid, sub_count FROM workset_parameter WHERE set_id = ", worksetId,
        sep=""
      )
    )

    for (wsParam in wsParamLst) {

      if (!wsParam$name %in% defRs$paramDic$parameter_name) {
        stop("parameter not found in model parameters list: ", wsParam$name)
      }

      paramRow <- defRs$paramDic[which(defRs$paramDic$parameter_name == wsParam$name), ]

      if (!paramRow$parameter_hid %in% setParamRs$parameter_hid) {
        stop("parameter is not in workset: ", wsParam$name)
      }
    }

    #
    # update parameters value and value notes
    #
    for (wsParam in wsParamLst) {

      # get parameter row
      paramRow <- defRs$paramDic[which(defRs$paramDic$parameter_name == wsParam$name), ]

      # get sub-value id
      wsRow <- setParamRs[which(setParamRs$parameter_hid == paramRow$parameter_hid), ]

      nSubId <- ifelse(!is.null(wsParam$subId) && !is.na(wsParam$subId), as.integer(wsParam$subId), 0L)
      # if (nSubId < 0 || nSubId >= wsRow$sub_count) stop("invalid sub-value id for parameter ", wsParam$name)

      # get name and size for each dimension if any dimensions exists for that parameter
      dimNames <- c("")
      colNames <- c("")
      dimLen <- c(0L)
      if (paramRow$parameter_rank > 0) {

        dimNames <- defRs$paramDims[which(defRs$paramDims$parameter_hid == paramRow$parameter_hid), "dim_name"]
        colNames <- defRs$paramDims[which(defRs$paramDims$parameter_hid == paramRow$parameter_hid), "col_name"]

        if (length(dimNames) != paramRow$parameter_rank) {
          stop("invalid length of dimension names vector for parameter: ", paramRow$parameter_name)
        }

        dimLen <- as.integer(
          table(defRs$typeEnum$type_hid)[as.character(
            defRs$paramDims[which(defRs$paramDims$parameter_hid == paramRow$parameter_hid), "type_hid"]
          )]
        )

        if (length(dimLen) != paramRow$parameter_rank) {
          stop("invalid length of dimension size vector for parameter: ", paramRow$parameter_name)
        }
      }

      # combine parameter definition to insert value and notes
      paramDef <- list(
        setId = worksetId,
        subId = nSubId,
        paramHid = paramRow$parameter_hid,
        dbTableName = paramRow$db_set_table,
        dims = data.frame(name = dimNames, dbName = colNames, size = dimLen, stringsAsFactors = FALSE)
      )

      # update parameter value
      updateWorksetParameterValue(dbCon, paramDef, wsParam$value)

      # if parameter value notes not empty then update value notes
      if (!is.null(wsParam$txt)) {
        updateWorksetParameterTxt(dbCon, paramDef, wsParam$txt)
      }
    }

    # change workset time of last update and reset readonly to read-write
    dbExecute(
      dbCon,
      paste(
        "UPDATE workset_lst",
        " SET is_readonly = 0, ",
        " update_dt = ", toQuoted(format(Sys.time(), "%Y-%m-%d %H:%M:%S")),
        " WHERE set_id = ", worksetId,
        sep=""
      )
    )
    isTrxCompleted <- TRUE; # completed OK
  },
  finally = {
    ifelse(isTrxCompleted, dbCommit(dbCon), dbRollback(dbCon))
  })
  return(ifelse(isTrxCompleted, worksetId, 0L))
}
