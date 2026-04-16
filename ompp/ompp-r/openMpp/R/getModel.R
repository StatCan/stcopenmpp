#' @title Find OpenM++ model definition in database
#' @description Find a model in the database and return its OpenM++ model definition, including input parameters and output tables.
#' @export
#' @param dbCon Database connection.
#' @param modelName Model name, e.g., `"modelOne"`.
#' @param modelDigest Optional model digest string, e.g., `"_201208171604590148_"`.
#'   If `NA` or `NULL`, the first model with minimum `model_id` is used.
#'
#' @details
#' Finds a model in the database and returns its OpenM++ model definition, including input parameters and output tables.
#'
#' Models can be identified by name and digest string (serving as the version).
#' If only one version of a model exists in the database, the `modelDigest` argument can be omitted.
#' If multiple models with the same name exist and `modelDigest` is `NA` or `NULL`, the model with the minimum `model_id` is returned.
#'
#' @return A list of database rows describing the model, including:
#' \describe{
#'   \item{modelDic}{`model_dic` row}
#'   \item{langLst}{list of languages (`lang_lst` table rows)}
#'   \item{typeDic}{`type_dic` rows of the model}
#'   \item{typeEnum}{`type_enum_lst` rows of the model, optional, can be empty}
#'   \item{paramDic}{`parameter_dic` rows of the model}
#'   \item{paramDims}{`parameter_dims` rows of the model, optional, can be empty}
#'   \item{tableDic}{`table_dic` rows of the model}
#'   \item{tableDims}{`table_dims` rows of the model, optional, can be empty}
#'   \item{tableAcc}{`table_acc` rows of the model}
#'   \item{tableExpr}{`table_expr` rows of the model}
#' }
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @author amc1999
#'
#' @note
#' To run examples, you must have the `modelOne` database `modelOne.sqlite` in the current directory.
#'
#' @seealso
#' \code{\link{getLanguages}}
#'
#' @examples
#' theDb <- dbConnect(RSQLite::SQLite(), "modelOne.sqlite", synchronous = "full")
#' invisible(dbGetQuery(theDb, "PRAGMA busy_timeout = 86400")) # recommended
#'
#' # Get model by name (single version)
#' defRs <- getModel(theDb, "modelOne")
#'
#' # Get model by name and digest (specific version)
#' def2012August17 <- getModel(theDb, "modelOne", "_201208171604590148_")
#'
#' dbDisconnect(theDb)

getModel <- function(dbCon, modelName, modelDigest = NA)
{
  # validate input parameters
  if (missing(dbCon)) stop("invalid (missing) database connection")
  if (is.null(dbCon) || !is(dbCon, "DBIConnection")) stop("invalid database connection")

  if (missing(modelName) || is.null(modelName) || !is.character(modelName)) {
    stop("invalid or empty model name")
  }

  # get model_dic row: single row expected
  sql <- ifelse(
    !missing(modelDigest) && !is.null(modelDigest) && !is.na(modelDigest),
    paste(
      "SELECT model_id, model_name, model_digest, model_type, model_ver, create_dt",
      " FROM model_dic",
      " WHERE model_name = ", toQuoted(modelName),
      " AND model_digest = ", toQuoted(modelDigest),
      " ORDER BY 1",
      sep=""
    ),
    paste(
      "SELECT model_id, model_name, model_digest, model_type, model_ver, create_dt",
      " FROM model_dic",
      " WHERE model_name = ", toQuoted(modelName),
      " AND model_id = ",
      " (",
      " SELECT MIN(MMD.model_id) FROM model_dic MMD WHERE MMD.model_name = " , toQuoted(modelName),
      " )",
      " ORDER BY 1",
      sep=""
    )
  )
  defRs <- list(modelDic = dbGetQuery(dbCon, sql))

  if (nrow(defRs$modelDic) != 1) stop("model not found: ", modelName, " ", modelDigest)

  # get languages
  defRs[["langLst"]] <- getLanguages(dbCon)

  # model types: type_dic and type_enum_lst
  defRs[["typeDic"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MT.model_id, MT.model_type_id, T.type_hid, T.type_name, T.dic_id",
      " FROM type_dic T",
      " INNER JOIN model_type_dic MT ON (MT.type_hid = T.type_hid AND MT.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2",
      sep = ""
    )
  )
  if (nrow(defRs$typeDic) <= 0) stop("no model types found")

  defRs[["typeEnum"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MT.model_id, MT.model_type_id, T.type_hid, T.enum_id, T.enum_name",
      " FROM type_enum_lst T",
      " INNER JOIN model_type_dic MT ON (MT.type_hid = T.type_hid AND MT.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2, 4",
      sep = ""
    )
  )
  # if (nrow(defRs$typeEnum) <= 0) stop("no model types found")

  # model parameter rows
  defRs[["paramDic"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT",
      " MP.model_id, MP.model_parameter_id, P.parameter_hid, P.parameter_name, P.db_run_table, P.db_set_table, P.parameter_rank, P.type_hid, MT.model_type_id",
      " FROM parameter_dic P",
      " INNER JOIN model_parameter_dic MP ON (MP.parameter_hid = P.parameter_hid AND MP.model_id = ", defRs$modelDic$model_id, ")",
      " INNER JOIN model_type_dic MT ON (MT.type_hid = P.type_hid AND MT.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2",
      sep = ""
    )
  )
  if (nrow(defRs$paramDic) <= 0) stop("Warning: parameters not found, nothing to do")

  # parameter dimensions rows: parameter_dims
  defRs[["paramDims"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT MP.model_id, MP.model_parameter_id, D.parameter_hid, D.dim_id, D.dim_name, D.type_hid, MT.model_type_id, D.dim_name AS col_name",
      " FROM parameter_dims D",
      " INNER JOIN model_parameter_dic MP ON (MP.parameter_hid = D.parameter_hid AND MP.model_id = ", defRs$modelDic$model_id, ")",
      " INNER JOIN model_type_dic MT ON (MT.type_hid = D.type_hid AND MT.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2, 4",
      sep = ""
    )
  )

  # for all parameters with dimensions:
  #   dimension must be in parameter_dims
  #   dimension type must be type_enum_lst
  if (!all(
      defRs$paramDic[which(defRs$paramDic$parameter_rank > 0), ]$parameter_hid %in%
      defRs$paramDims$parameter_hid
    )) {
    stop("parameter dimension(s) not found")
  }
  if (!all(defRs$paramDims$type_hid %in% defRs$typeEnum$type_hid)) {
    stop("parameter dimension type(s) not found")
  }

  # set dimension db colunm name as "dim" + id
  if (nrow(defRs$paramDims) > 0) {
    defRs$paramDims$col_name <- paste("dim", defRs$paramDims$dim_id, sep = "")
  }

  # model output tables rows: table_dic
  defRs[["tableDic"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT M.model_id, M.model_table_id, T.table_hid, T.table_name, T.db_expr_table, T.db_acc_table, T.table_rank, T.is_sparse",
      " FROM table_dic T",
      " INNER JOIN model_table_dic M ON (M.table_hid = T.table_hid AND M.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2",
      sep = ""
    )
  )
  if (nrow(defRs$tableDic) <= 0) stop("Warning: output tables not found, nothing to do")

  # dimensions of model output tables rows: table_dims
  defRs[["tableDims"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT M.model_id, M.model_table_id, D.table_hid, D.dim_id, D.dim_name, D.type_hid, MT.model_type_id, D.is_total, D.dim_size, D.dim_name AS col_name",
      " FROM table_dims D",
      " INNER JOIN model_table_dic M ON (M.table_hid = D.table_hid AND M.model_id = ", defRs$modelDic$model_id, ")",
      " INNER JOIN model_type_dic MT ON (MT.type_hid = D.type_hid AND MT.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2, 4",
      sep = ""
    )
  )

  # for all output tables with dimensions:
  #   dimension must be in table_dims
  #   dimension type must be type_enum_lst
  if (!all(
      defRs$tableDic[which(defRs$tableDic$table_rank > 0), ]$table_hid %in%
      defRs$tableDims$table_hid
    )) {
    stop("output table dimension(s) not found")
  }
  if (!all(defRs$tableDims$type_hid %in% defRs$typeEnum$type_hid)) {
    stop("output table dimension(s) not found")
  }

  # set dimension db colunm name as "dim" + id
  if (nrow(defRs$tableDims) > 0) {
    defRs$tableDims$col_name <- paste("dim", defRs$tableDims$dim_id, sep = "")
  }

  # output table accumulators rows: table_acc
  defRs[["tableAcc"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT M.model_id, M.model_table_id, A.table_hid, A.acc_id, A.acc_name, A.is_derived, A.acc_name AS col_name",
      " FROM table_acc A",
      " INNER JOIN model_table_dic M ON (M.table_hid = A.table_hid AND M.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2, 4",
      sep = ""
    )
  )
  if (nrow(defRs$tableAcc) <= 0) stop("definition for output table(s) accumulators not found")

  # set accumulator db colunm name: (is derived accumulator) ? "acc" + id : "expr" + id
  defRs$tableAcc$col_name <- paste(
    ifelse(defRs$tableAcc$is_derived == 0, "acc", "expr"),
    defRs$tableAcc$acc_id,
    sep = ""
  )

  # output table expressions (analysis dimension items) rows: table_expr
  defRs[["tableExpr"]] <- dbGetQuery(
    dbCon,
    paste(
      "SELECT M.model_id, M.model_table_id, E.table_hid, E.expr_id, E.expr_name, E.expr_decimals, E.expr_name AS col_name",
      " FROM table_expr E",
      " INNER JOIN model_table_dic M ON (M.table_hid = E.table_hid AND M.model_id = ", defRs$modelDic$model_id, ")",
      " ORDER BY 1, 2, 4",
      sep = ""
    )
  )
  if (nrow(defRs$tableExpr) <= 0) stop("definition for output table(s) expressions not found")

  # set expression db colunm name as "expr" + id
  defRs$tableExpr$col_name <- paste("expr", defRs$tableExpr$expr_id, sep = "")

  return(defRs)
}
