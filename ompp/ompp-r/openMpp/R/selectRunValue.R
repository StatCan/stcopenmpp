#' @title Select output table expression values from model run result
#' @description nRetrieve output table expression values from a specific model run result.
#' @export
#' @param dbCon Database connection.
#' @param defRs Model definition: database rows describing model input parameters and output tables.
#' @param runId Positive integer. ID of model run results.
#' @param tableName Output table name (character), e.g., `"sexAge"`.
#' @param exprName Optional output table expression name (character), e.g., `"AverageAge"`.
#'                 If missing or `NA`, returns all expressions.
#'
#' @details
#' Returns values of output table expressions from the model run results with the specified `runId`.
#' You can retrieve either a single expression by specifying its name or all expressions if
#' `exprName` is missing. The `defRs` object must be obtained via `\code{\link{getModel}}`.
#'
#' @return
#' Data frame of database rows with output table expression(s) dimensions and values:
#' \describe{
#'   \item{expr_id}{Expression number (zero-based).}
#'   \item{dim0,...,dimN}{Dimension items enum IDs (not returned if rank is zero).}
#'   \item{value}{Output table expression value.}
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
#' \code{\link{selectRunAccumulator}}, \code{\link{selectRunParameter}}
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
#' # select output table expression(s) value
#' expr2_ValueRs <- selectRunOutputValue(theDb, defRs, runId, "salarySex", "expr2")
#' allExprValueRs <- selectRunOutputValue(theDb, defRs, runId, "salarySex")
#'
#' dbDisconnect(theDb)

selectRunOutputValue <- function(dbCon, defRs, runId, tableName, exprName = NA)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(defRs)) stop("invalid (missing) model definition")
  if (is.null(defRs) || any(is.na(defRs)) || !is.list(defRs)) stop("invalid or empty model definition")

  if (missing(runId)) stop("invalid (missing) run id")
  if (is.null(runId) || is.na(runId) || !is.integer(runId)) stop("invalid or empty run id")
  if (runId <= 0L) stop("run id must be positive: ", runId)

  if (missing(tableName)) stop("invalid (missing) output table name")
  if (is.null(tableName)) stop("invalid or empty output table name")
  if (is.na(tableName) || !is.character(tableName) || length(tableName) <= 0) {
    stop("invalid or empty output table name")
  }

  # get table row by name
  tableRow <- defRs$tableDic[which(defRs$tableDic$table_name == tableName), ]
  if (nrow(tableRow) != 1) {
    stop("output table not found in model output tables list: ", tableName)
  }

  nRank <- tableRow$table_rank
  dbTableName <- tableRow$db_expr_table

  # find output expression id by name, if specified
  exprId <- NA
  if (!missing(exprName) && !is.na(exprName)) {
    if (!is.character(exprName) || length(exprName) <= 0) stop("invalid or empty output expression name")

    uRow <- defRs$tableExpr[which(defRs$tableExpr$table_hid == tableRow$table_hid & defRs$tableExpr$expr_name == exprName), ]

    if (nrow(uRow) != 1) {
      stop("output table ", tableName, " does not contain expression: ", exprName)
    }
    exprId <- uRow$expr_id
  }

  # check if run id is belong the model and completed
  rlRow <- dbGetQuery(
    dbCon,
    paste(
      "SELECT run_id FROM run_lst",
      " WHERE run_id = ", runId,
      " AND model_id = ", defRs$modelDic$model_id,
      " AND status = 's'",
      sep=""
    )
  )
  if (nrow(rlRow) != 1L) {
    stop("model run results not found (or not completed), run id: ", runId)
  }

  # SELECT expr_id, dim0, dim1, expr_value
  # FROM salarySex_v2012_820
  # WHERE run_id =
  # (
  #   SELECT base_run_id FROM run_table WHERE run_id = 2 AND table_hid = 12345
  # )
  # AND expr_id = 3
  # ORDER BY 1, 2, 3
  #
  d <- defRs$tableDims[which(defRs$tableDims$table_hid == tableRow$table_hid), c("col_name", "dim_name")]

  sqlSel <-
    paste(
      "SELECT expr_id, ",
      ifelse(nRank > 0L,
        paste(
          paste(d[,1], " AS ", '"', d[,2], '"', sep = "", collapse = ", "),
          ", ",
          sep = ""
        ),
        ""
      ),
      " expr_value",
      " FROM ", dbTableName,
      " WHERE run_id = ",
      " (",
      " SELECT base_run_id FROM run_table WHERE run_id = ", runId, " AND table_hid = ", tableRow$table_hid,
      " )",
      ifelse(!is.na(exprId),
        paste(" AND expr_id = ", exprId, sep = ""),
        ""
      ),
      " ORDER BY 1",
      ifelse(nRank > 0L,
        paste(", ",
          paste(2L:(nRank + 1L), sep = "", collapse = ", "),
          sep = ""
        ),
        ""
      ),
      sep = ""
    )

  selRs <- dbGetQuery(dbCon, sqlSel)
  return(selRs)
}
