#' @title Create new working set of model parameters
#' @description Create a new workset containing a **full set of model parameters**.
#' The workset must include **all model parameters**.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param setDef Optional workset description as a data frame. Columns:
#'   \describe{
#'     \item{name}{Workset name (must be unique).}
#'     \item{lang}{Language code.}
#'     \item{descr}{Workset description.}
#'     \item{note}{Optional workset notes.}
#'   }
#' @param ... List of parameter values and optional value notes.
#'   Each element is a list with components:
#'   \describe{
#'     \item{name}{Parameter name (character).}
#'     \item{subCount}{Optional: number of parameter sub-values (default: 1).}
#'     \item{defaultSubId}{Optional: default sub-value ID (default: 0).}
#'     \item{subId}{Optional: parameter sub-value ID (default: 0).}
#'     \item{value}{Parameter value: scalar, vector, or data frame.
#'       If a data frame, it must contain columns \code{dimName0}, \code{dimName1}, …, and \code{value},
#'       and each column length must equal the product of dimension sizes.}
#'     \item{txt}{Optional workset parameter text: a data frame with columns
#'       \code{lang} (language code) and \code{note} (value notes).}
#'   }
#'
#' @details
#' This function creates a new working set containing **all input parameters** of a model.
#' A workset can be:
#' \itemize{
#'   \item Full: includes all model parameters.
#'   \item Subset: includes only some parameters (must be based on an existing model run, see \code{\link{createWorksetBasedOnRun}}).
#' }
#'
#' Each model has a **default workset**: the first workset with \code{set_id = min(set_id)}, which always includes all parameters.
#'
#' To create a new full workset, pass **all model parameters** through \code{...}.
#' To create a subset, use \code{\link{createWorksetBasedOnRun}} with a previous run.
#'
#' Each workset has a **unique set ID** (positive integer) and a **unique name**.
#' Use \code{\link{getWorksetIdByName}} to find a workset by name.
#'
#' Worksets must be **read-only** to run the model, so call \code{\link{setReadonlyWorkset}} after creating the workset.
#'
#' Parameters may have multiple sub-values (default: 1).
#' Use \code{subCount > 1} to define multiple sub-values.
#' Specify sub-value ID using \code{subId} (default: 0) and the default sub-value ID using \code{defaultSubId} (default: 0).
#'
#' Use \code{\link{getModel}} to obtain the model definition (\code{defRs}).
#'
#' @return Integer ID of the new workset, or 0L on error.
#'
#' @references OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note To run examples, you must have the \code{modelOne} database (\code{modelOne.sqlite}) in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getDefaultWorksetId}}, \code{\link{getWorksetIdByName}},
#' \code{\link{createWorksetBasedOnRun}}, \code{\link{copyWorksetParameterFromRun}},
#' \code{\link{setReadonlyWorkset}}, \code{\link{setReadonlyDefaultWorkset}},
#' \code{\link{updateWorksetParameter}}
#'
#' @examples
#' # Model parameters:
#' #   age by sex: double[4, 2]
#' #   salary by age: int[3, 4]
#' #   salary level: int enum[3]
#' #   base salary: int enum scalar
#' #   starting seed: int scalar
#' #   file path: string
#'
#' # Example parameter values
#' ageSex <- list(
#'   name = "ageSex", subCount = 1L, defaultSubId = 0L, subId = 0L,
#'   value = c(10, rep(c(1,2,3), times = 2), 20),
#'   txt = data.frame(lang = c("EN", "FR"), note = c("age by sex value notes", NA), stringsAsFactors = FALSE)
#' )
#' salaryAge <- list(
#'   name = "salaryAge",
#'   value = c(100L, rep(c(10L,20L,30L), times = 3), 200L, 300L),
#'   txt = data.frame(lang = c("EN", "FR"), note = c("salary by age value notes","FR salary by age value notes"), stringsAsFactors = FALSE)
#' )
#' salaryFull <- list(
#'   name = "salaryFull", value = c(33L,33L,22L),
#'   txt = data.frame(lang = c("EN"), note = c("salary level for full or part time job"), stringsAsFactors = FALSE)
#' )
#' baseSalary <- list(
#'   name = "baseSalary", value = 22L,
#'   txt = data.frame(lang = c("EN"), note = c("base salary notes"), stringsAsFactors = FALSE)
#' )
#' startingSeed <- list(
#'   name = "StartingSeed", value = 127L,
#'   txt = data.frame(lang = c("EN"), note = c("random generator starting seed"), stringsAsFactors = FALSE)
#' )
#' isOldAge <- list(
#'   name = "isOldAge", value = c(TRUE, FALSE, TRUE, FALSE),
#'   txt = data.frame(lang = c("EN"), note = c("Is Old Age notes"), stringsAsFactors = FALSE)
#' )
#' filePath <- list(
#'   name = "filePath", value = "file R path",
#'   txt = data.frame(lang = c("EN"), note = c("file path string parameter"), stringsAsFactors = FALSE)
#' )
#'
#' # Workset definition
#' inputSet <- data.frame(
#'   name = "myData",
#'   lang = c("EN", "FR"),
#'   descr = c("full set of parameters", "description in French"),
#'   note = c("full set of parameters notes", NA),
#'   stringsAsFactors = FALSE
#' )
#'
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400"))
#'
#' defRs <- getModel(theDb, "modelOne")
#'
#' setId <- createWorkset(theDb, defRs, inputSet, ageSex, salaryAge, salaryFull, baseSalary, startingSeed, isOldAge, filePath)
#' if (setId <= 0L) stop("workset creation failed")
#'
#' setReadonlyWorkset(theDb, defRs, TRUE, setId)
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

createWorkset <- function(dbCon, defRs, setDef, ...)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  # create new workset
  setId <- createNewWorkset(dbCon, defRs, FALSE, NA, setDef, ...)
  return(setId)
}

#' @title Create new working set of model parameters based on previous run
#' @description Create a new workset as a **subset of model parameters** based on parameters from an existing model run.
#' New parameter values can also be supplied via `...`.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param baseRunId ID of the model run results to base the workset on (must be positive integer).
#' @param setDef Optional workset description as a data frame. Columns:
#'   \describe{
#'     \item{name}{Workset name (must be unique).}
#'     \item{lang}{Language code.}
#'     \item{descr}{Workset description.}
#'     \item{note}{Optional workset notes.}
#'   }
#' @param ... List of parameter values and optional value notes.
#'   Each element is a list with components:
#'   \describe{
#'     \item{name}{Parameter name (character).}
#'     \item{subCount}{Optional: number of parameter sub-values (default: 1).}
#'     \item{defaultSubId}{Optional: default sub-value ID (default: 0).}
#'     \item{subId}{Optional: parameter sub-value ID (default: 0).}
#'     \item{value}{Parameter value: scalar, vector, or data frame.
#'       If a data frame, it must have columns \code{dim0}, \code{dim1}, …, and \code{value},
#'       and each column length must equal the product of dimension sizes.}
#'     \item{txt}{Optional workset parameter text: data frame with columns
#'       \code{lang} (language code) and \code{note} (value notes).}
#'   }
#'
#' @details
#' This function creates a new workset by combining:
#' \itemize{
#'   \item Parameters from an existing model run.
#'   \item New or modified parameters supplied through `...`.
#' }
#'
#' Worksets can be:
#' \itemize{
#'   \item Full: contains all model parameters.
#'   \item Subset: contains only some parameters (must be based on an existing run).
#' }
#'
#' Each model has a **default workset**: the first workset (\code{set_id = min(set_id)}) which always includes all parameters.
#'
#' To create a **full workset**, pass all model parameters via `...`.
#' To create a **subset**, use this function with an existing run and supply only the parameters you want to change.
#'
#' Each workset has a **unique set ID** and may have a unique name.
#' Use \code{\link{getWorksetIdByName}} to find a workset by name.
#'
#' Worksets must be **read-only** to run the model; call \code{\link{setReadonlyWorkset}} after creating the workset.
#'
#' Parameters may have multiple sub-values (default: 1).
#' Use \code{subCount > 1} to define multiple sub-values.
#' Specify sub-value ID using \code{subId} (default: 0) and default sub-value ID using \code{defaultSubId} (default: 0).
#'
#' Use \code{\link{getModel}} to obtain the model definition (\code{defRs}).
#'
#' @return Integer ID of the new workset, or 0L on error.
#'
#' @references OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note To run examples, you must have the \code{modelOne} database (\code{modelOne.sqlite}) in the current directory.
#'
#' @seealso
#' \code{\link{getModel}}, \code{\link{getFirstRunId}}, \code{\link{getLastRunId}},
#' \code{\link{getDefaultWorksetId}}, \code{\link{getWorksetIdByName}},
#' \code{\link{createWorkset}}, \code{\link{copyWorksetParameterFromRun}},
#' \code{\link{setReadonlyWorkset}}, \code{\link{setReadonlyDefaultWorkset}},
#' \code{\link{updateWorksetParameter}}
#'
#' @examples
#' # Example: StartingSeed parameter with two sub-values
#' seedSubVal <- list(
#'   name = "StartingSeed",
#'   subCount = 2L,        # two sub-values
#'   defaultSubId = 1L,    # default sub-value id
#'   subId = 0L,           # sub-value id 0
#'   value = 100L          # sub-value 0
#' )
#'
#' # Workset definition
#' inputSet <- data.frame(
#'   name = "myOtherData",
#'   lang = "EN",
#'   descr = "new set of parameters",
#'   note = "new set of parameters with updated salary by age",
#'   stringsAsFactors = FALSE
#' )
#'
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400"))
#' defRs <- getModel(theDb, "modelOne")
#'
#' # Create workset: new value for StartingSeed, all other parameters from run 101
#' setId <- createWorksetBasedOnRun(theDb, defRs, 101L, inputSet, seedSubVal)
#' if (setId <= 0L) stop("workset creation failed")
#'
#' # Add seed parameter sub-value 1
#' seedSubVal <- list(name = "StartingSeed", subId = 1L, value = 200L)
#' setId <- updateWorksetParameter(theDb, defRs, setId, seedSubVal)
#' if (setId <= 0L) stop("failed to update workset parameter")
#'
#' # Copy ageSex parameter from run 102
#' copyWorksetParameterFromRun(theDb, defRs, setId, 102L, list(name = "ageSex"))
#'
#' # Make workset read-only
#' setReadonlyWorkset(theDb, defRs, TRUE, setId)
#' dbDisconnect(theDb)
#'
#' @keywords OpenM++ database

createWorksetBasedOnRun <-  function(dbCon, defRs, baseRunId, setDef, ...)
  {
    # validate input parameters
    if (missing(dbCon)) stop("invalid (missing) database connection")
    if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

    if (missing(defRs)) stop("invalid (missing) model definition")
    if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

    if (missing(baseRunId)) stop("invalid (missing) base run id")
    if (is.null(baseRunId)) stop("invalid (missing) base run id")
    if (is.na(baseRunId) || !is.integer(baseRunId) || baseRunId <= 0L) {
      stop("invalid (missing) base run id: ", baseRunId)
    }

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

    # create new workset
    setId <- createNewWorkset(dbCon, defRs, TRUE, baseRunId, setDef, ...)
    return(setId)
  }

#' @title Create a new working set of model parameters (internal use)
#' @description **Internal function. Must be called within a transaction scope.**
#' Creates a new working set of model parameters and returns its ID.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param i_isRunBased Logical; if TRUE, use parameters from a base run (`i_baseRunId`).
#'   If FALSE, all parameters must be supplied via `...`.
#' @param i_baseRunId ID of the model run results to copy parameters from (used if `i_isRunBased = TRUE`).
#' @param i_setDef Optional workset definition as a data frame with columns:
#'   \describe{
#'     \item{name}{Working set name.}
#'     \item{lang}{Language code.}
#'     \item{descr}{Workset description.}
#'     \item{note}{Optional workset notes.}
#'   }
#' @param ... List of parameter values and optional value notes. Each element is a list with components:
#'   \describe{
#'     \item{name}{Parameter name (character).}
#'     \item{subCount}{Optional: number of parameter sub-values (default: 1).}
#'     \item{defaultSubId}{Optional: default sub-value ID (default: 0).}
#'     \item{subId}{Optional: sub-value ID (default: 0).}
#'     \item{value}{Parameter value: scalar, vector, or data frame.
#'       If a data frame, it must have columns \code{dim0}, \code{dim1}, …, \code{value},
#'       and each column length must equal the product of dimension sizes.}
#'     \item{txt}{Optional workset parameter text: data frame with columns \code{lang} (language code) and \code{note} (value notes).}
#'   }
#'
#' @details
#' This function creates a new workset of model parameters.
#' If `i_isRunBased = TRUE`, parameters are copied from an existing run (`i_baseRunId`).
#' Otherwise, all parameters must be supplied via `...`.
#'
#' Each workset has a unique set ID (positive integer) and can optionally have a name.
#' Parameters may have multiple sub-values; defaults are `subCount = 1`, `subId = 0`, `defaultSubId = 0`.
#'
#' **Note:** This function is internal and must be executed in a transaction. Do not call directly from outside code unless you understand transaction scope.
#'
#' @return Integer ID of the newly created workset. Returns `<= 0` on error.
#'
#' @keywords internal

createNewWorkset <-  function(dbCon, defRs, i_isRunBased, i_baseRunId, i_setDef, ...)
  {
    # validate input parameters
    if (missing(dbCon)) stop("invalid (missing) database connection")
    if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

    if (missing(defRs)) stop("invalid (missing) model definition")
    if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

    # get list of languages and validate parameters data
    wsParamLst <- list(...)
    if (!i_isRunBased && length(wsParamLst) <= 0) stop("invalid (missing) workset parameters list")

    if (length(wsParamLst) > 0 && !validateParameterValueLst(defRs$langLst, TRUE, wsParamLst)) return(0L)

    # validate workset text
    isAnyWsTxt <- validateTxtFrame("workset text", defRs$langLst, i_setDef)

    # check if supplied parameters are in model: parameter_name in parameter_dic table
    # if all parameters required then check if ALL parameters supplied
    if (!i_isRunBased) nameLst <- defRs$paramDic$parameter_name

    for (wsParam in wsParamLst) {
      if (!wsParam$name %in% defRs$paramDic$parameter_name) {
        stop("parameter not found in model parameters list: ", wsParam$name)
      }
      if (!i_isRunBased) nameLst <- nameLst[which(nameLst != wsParam$name)]
    }

    if (!i_isRunBased && length(nameLst) > 0) {
      stop(
        "workset must contain ALL model parameters,",
        " not found: ", length(nameLst), " parameter(s),",
        " first 10 names: ", paste(head(nameLst, n = 10), collapse = ", ")
      )
    }

    # execute in transaction scope
    setId <- 0L

    isTrxCompleted <- FALSE;
    tryCatch({
      dbBegin(dbCon)

      # get next set id
      dbExecute(dbCon, "UPDATE id_lst SET id_value = id_value + 1 WHERE id_key = 'run_id_set_id'")
      idRs <- dbGetQuery(dbCon, "SELECT id_value FROM id_lst WHERE id_key = 'run_id_set_id'")
      if (nrow(idRs) <= 0L || idRs$id_value <= 0L) stop("can not get new set id from id_lst table")

      setId <- idRs$id_value

      # workset name, make auto-name if empty
      setName <- ifelse(isAnyWsTxt, i_setDef$name, NA)
      if (is.na(setName)) setName <- paste("set_", setId, sep = "")

      # create workset
      dbExecute(
        dbCon,
        paste(
          "INSERT INTO workset_lst (set_id, base_run_id, model_id, set_name, is_readonly, update_dt)",
          " VALUES (",
          setId, ", ",
          ifelse(i_isRunBased, i_baseRunId, "NULL"), ", ",
          defRs$modelDic$model_id, ", ",
          toQuoted(setName), ", ",
          " 0, ",
          toQuoted(format(Sys.time(), "%Y-%m-%d %H:%M:%S")),
          " )",
          sep = ""
        )
      )

      # insert workset text rows where laguage and description non-empty
      if (isAnyWsTxt) {
        sqlInsTxt <-
          paste(
            "INSERT INTO workset_txt (set_id, lang_id, descr, note)",
            " SELECT",
            " W.set_id,",
            " (SELECT L.lang_id FROM lang_lst L WHERE L.lang_code = :lang),",
            " :descr,",
            " :note",
            " FROM workset_lst W WHERE W.set_id = ", setId,
            sep = ""
          )
        dbExecute(
          dbCon,
          sqlInsTxt,
          params = subset(i_setDef, !is.na(lang) & !is.na(descr), select = c(lang, descr, note))
        )
      }

      #
      # update parameters value and value notes
      #
      for (wsParam in wsParamLst) {

        # get parameter row
        paramRow <- defRs$paramDic[which(defRs$paramDic$parameter_name == wsParam$name), ]

        # validate sub-value count
        isCount <- !is.null(wsParam$subCount) && !is.na(wsParam$subCount)
        nCount <- ifelse(isCount, as.integer(wsParam$subCount), 1L)
        if (nCount < 1) stop("invalid (less than 1) sub-value count for parameter ", wsParam$name)

        if (i_isRunBased) {
          rpRs <- dbGetQuery(
            dbCon,
            paste(
              "SELECT sub_count FROM run_parameter ",
              " WHERE run_id = ",
              " (",
              " SELECT base_run_id FROM run_parameter WHERE run_id = ", i_baseRunId, " AND parameter_hid = ", paramRow$parameter_hid,
              " )",
              sep = ""
            )
          )
          if (nrow(rpRs) == 1L) {
            if (!isCount){
              nCount <- rpRs$sub_count
            }
            else {
              if (nCount > rpRs$sub_count) stop("invalid (less than 1) sub-value count for parameter ", wsParam$name)
            }
          }
        }

        # default sub-value id for that parameter and sub-value id for values
        nDefaultId <- ifelse(!is.null(wsParam$defaultSubId) && !is.na(wsParam$defaultSubId), as.integer(wsParam$defaultSubId), 0L)
        nSubId <- ifelse(!is.null(wsParam$subId) && !is.na(wsParam$subId), as.integer(wsParam$subId), 0L)
        # if (nSubId < 0 || nSubId >= nCount) stop("invalid sub-value id for parameter ", wsParam$name)

        # add parameter into workset
        dbExecute(
          dbCon,
          paste(
            "INSERT INTO workset_parameter (set_id, parameter_hid, sub_count, default_sub_id) VALUES (",
            setId, ", ",
            paramRow$parameter_hid, ", ",
            nCount, ", ",
            nDefaultId, " )",
            sep = ""
          )
        )

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
          setId = setId,
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

      isTrxCompleted <- TRUE; # completed OK
    },
    finally = {
      ifelse(isTrxCompleted, dbCommit(dbCon), dbRollback(dbCon))
    })
    return(ifelse(isTrxCompleted, setId, 0L))
  }
