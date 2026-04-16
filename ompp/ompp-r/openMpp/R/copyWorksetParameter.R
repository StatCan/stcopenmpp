#' @title Copy parameters to working set from existing model run
#' @description Copy parameters to a working set from an existing model run, optionally adding new value notes.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param worksetId ID of the parameters working set (must be a positive integer).
#' @param baseRunId ID of the model run results (must be a positive integer).
#' @param ... List of parameters with values and optional value notes.
#'   Each element is a list with components:
#'   \describe{
#'     \item{name}{Parameter name (character).}
#'     \item{subCount}{Optional: sub-value count (default: 1).}
#'     \item{subId}{Optional: sub-value ID (default: 0).}
#'     \item{value}{Parameter value. Can be scalar, vector, or data frame.}
#'     \item{txt}{Optional workset parameter text: a data frame with columns
#'       \code{lang} (language code) and \code{note} (value notes).}
#'   }
#'
#' @details
#' This function allows you to copy input parameter values from an existing model run into a working set
#' and optionally update their value notes. Parameters can have multiple sub-values (default is one).
#' All sub-values are copied from the existing run.
#'
#' A workset is a collection of model parameters. It may be:
#' \itemize{
#'   \item Full: includes all model parameters.
#'   \item Subset: includes only some parameters (must be based on an existing model run).
#' }
#'
#' Each model has a default workset (the first workset for the model, \code{set_id = min(set_id)}),
#' which always includes all model parameters.
#'
#' To create a new full workset, pass all model parameters into \code{\link{createWorkset}}.
#' To create a subset based on a previous run, use \code{\link{createWorksetBasedOnRun}}.
#'
#' Worksets must be non-read-only to add parameters using this function, and read-only to run the model.
#' Typically, wrap this function with \code{\link{setReadonlyWorkset}} calls.
#'
#' Use \code{\link{getModel}} to obtain \code{defRs}.
#'
#' @return Integer ID of the working set, or 0L on error.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples, you must have the \code{modelOne} database (\code{modelOne.sqlite}) in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getFirstRunId}}, \code{\link{getLastRunId}},
#' \code{\link{getDefaultWorksetId}}, \code{\link{getWorksetIdByName}},
#' \code{\link{createWorkset}}, \code{\link{createWorksetBasedOnRun}},
#' \code{\link{setReadonlyWorkset}}, \code{\link{setReadonlyDefaultWorkset}},
#' \code{\link{updateWorksetParameter}}
#'
#' @examples
#' # Model parameters:
#' #   age by sex: double[4, 2]
#' #   salary by age: int[3, 4]
#' #   starting seed: integer
#'
#' # Name, description, and notes for this parameter set
#' inputSet <- data.frame(
#'   name = "myOtherData",
#'   lang = "EN",
#'   descr = "new set of parameters",
#'   note = "new set of parameters with updated salary by age",
#'   stringsAsFactors = FALSE
#' )
#'
#' # Age by sex parameter value notes
#' ageSex <- list(
#'   name = "ageSex",
#'   txt = data.frame(
#'     lang = c("EN", "FR"),
#'     note = c("age by sex value notes", NA),
#'     stringsAsFactors = FALSE
#'   )
#' )
#'
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # Get model definition
#' defRs <- getModel(theDb, "modelOne")
#'
#' # Create new working set based on previous run with ID = 101
#' setId <- createWorksetBasedOnRun(theDb, defRs, 101L, inputSet)
#' if (setId <= 0L) stop("workset creation failed: ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
#'
#' # Copy ageSex parameter value from run ID = 102
#' copyWorksetParameterFromRun(theDb, defRs, setId, 102L, ageSex)
#'
#' # Make workset read-only to run the model
#' setReadonlyWorkset(theDb, defRs, TRUE, setId)
#'
#' dbDisconnect(theDb)
#'
#' # You can now run the model with the updated ageSex parameter value
#'
#' @keywords OpenM++ database

copyWorksetParameterFromRun <- function(dbCon, defRs, worksetId, baseRunId, ...)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  if (missing(worksetId)) stop("invalid (missing) workset id")
  if (is.null(worksetId) || is.na(worksetId) || !is.integer(worksetId)) stop("invalid or empty workset id")
  if (worksetId <= 0L) stop("workset id must be positive: ", worksetId)

  if (missing(baseRunId)) stop("invalid (missing) base run id")
  if (is.null(baseRunId)) stop("invalid (missing) base run id")
  if (is.na(baseRunId) || !is.integer(baseRunId) || baseRunId <= 0L) {
    stop("invalid (missing) base run id: ", baseRunId)
  }

  # get list of languages and validate parameters data
  wsParamLst <- list(...)
  if (length(wsParamLst) <= 0) stop("invalid (missing) workset parameters list")

  if (!validateParameterValueLst(defRs$langLst, FALSE, wsParamLst)) return(0L)

  # check if base run id is belong to the model
  runRs <- dbGetQuery(
    dbCon,
    paste(
      "SELECT run_id, model_id FROM run_lst WHERE run_id = ", baseRunId,
      sep = ""
    )
  )
  if (nrow(runRs) != 1L || runRs$model_id != defRs$modelDic$model_id) {
    stop("base run id not found: ", baseRunId, " or not belong to model: ", defRs$modelDic$model_name, " ", defRs$modelDic$model_digest)
  }

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
    # check if supplied parameters are not already in workset_parameter table
    setParamRs <- dbGetQuery(
      dbCon,
      paste(
        "SELECT set_id, parameter_hid FROM workset_parameter WHERE set_id = ", worksetId,
        sep=""
      )
    )

    for (wsParam in wsParamLst) {

      if (!wsParam$name %in% defRs$paramDic$parameter_name) {
        stop("parameter not found in model parameters list: ", wsParam$name)
      }

      paramRow <- defRs$paramDic[which(defRs$paramDic$parameter_name == wsParam$name), ]

      if (paramRow$parameter_hid %in% setParamRs$parameter_hid) {
        stop("parameter already in workset: ", wsParam$name)
      }
    }

    #
    # copy parameters value and update value notes
    #
    for (wsParam in wsParamLst) {

      # get parameter row
      paramRow <- defRs$paramDic[which(defRs$paramDic$parameter_name == wsParam$name), ]

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
        paramHid = paramRow$parameter_hid,
        dbTableName = paramRow$db_set_table,
        dims = data.frame(name = dimNames, dbName = colNames, size = dimLen, stringsAsFactors = FALSE)
      )

      # add parameter into workset
      dbExecute(
        dbCon,
        paste(
          "INSERT INTO workset_parameter (set_id, parameter_hid, sub_count, default_sub_id)",
          " SELECT ", worksetId, ", ", paramDef$paramHid, ", sub_count, 0",
          " FROM run_parameter",
          " WHERE parameter_hid = ", paramDef$paramHid,
          " AND run_id = ",
          " (",
          " SELECT base_run_id FROM run_parameter WHERE run_id = ", baseRunId, " AND parameter_hid = ", paramDef$paramHid,
          " )",
          sep = ""
        )
      )

      # copy parameter values from run results table into workset table
      # use base run id if run is not a full run and parameter value(s) stored under base run id
      nameCs <- ifelse(
        paramRow$parameter_rank > 0,
        paste(paste(paramDef$dims$dbName, sep = "", collapse = ", "), ", ", sep = ""),
        ""
      )

      dbExecute(
        dbCon,
        paste(
          "INSERT INTO ", paramDef$dbTableName,
          " (set_id, sub_id, ", nameCs, " param_value)",
          " SELECT ",
          worksetId, ", sub_id, ", nameCs, " param_value",
          " FROM ", paramRow$db_run_table,
          " WHERE run_id = ",
          " (",
          " SELECT base_run_id FROM run_parameter WHERE run_id = ", baseRunId, " AND parameter_hid = ", paramDef$paramHid,
          " )",
          sep = ""
        )
      )

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
