#' @title Return SQL-quoted string
#' @description Return a SQL-quoted string. For example, `O'Connor` becomes `'O''Connor'`.
#' @export
#' @param srcStr Source string to be SQL-quoted.
#'
#' @details
#' This function duplicates each apostrophe in the input string and encloses the result in single quotes.
#' For example, `O'Connor` becomes `'O''Connor'`.
#'
#' @return SQL-quoted string suitable for use in SQL statements.
#'
#' @references
#' OpenM++ documentation: \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @seealso
#' \url{https://github.com/openmpp/openmpp.github.io/wiki}
#'
#' @examples
#' someName <- toQuoted("O'Connor")
#' # someName
#' # [1] "'O''Connor'"
#'
#' sql <- paste(
#'   "SELECT * FROM someTable WHERE name = ", toQuoted("O'Connor"), sep=""
#' )
#' # sql
#' # [1] "SELECT * FROM someTable WHERE name = 'O''Connor'"
#'
#' @keywords OpenM++ database

toQuoted <- function(srcStr)
{
  paste(
    "'", gsub("'", "''", srcStr, fixed = TRUE), "'",
    sep = ""
  )
}

#' @title Validate text parameter (workset or task text)
#' @description Internal function to validate a text parameter. Checks that required columns exist
#' and that the data frame contains valid rows. Stops execution if invalid,
#' returns `FALSE` if the input is empty, or `TRUE` if any valid data is present.
#' @export
#' @param i_msgPart Character. Part of the message used in error messages.
#' @param i_langRs Data frame. Rows of `lang_lst` table (available languages).
#' @param i_txt Optional data frame with text data. Expected columns:
#'   \describe{
#'     \item{name}{Name of the workset or task}
#'     \item{lang}{Language code}
#'     \item{descr}{Description of the workset or task}
#'     \item{note}{Optional notes for the workset or task}
#'   }
#'
#' @details
#' This function is intended for internal use only. It validates the structure and content
#' of workset or task text parameters. Execution stops with an error if invalid data is found.
#'
#' @return
#' Logical:
#' \describe{
#'   \item{TRUE}{If `i_txt` contains any valid data}
#'   \item{FALSE}{If `i_txt` is empty}
#' }
#'
#' @keywords internal

validateTxtFrame <- function(i_msgPart, i_langRs, i_txt)
{
  # validate data frame itself, exit if empty
  if (missing(i_txt)) return(FALSE)
  if (is.null(i_txt)) return(FALSE)
  if (!is.data.frame(i_txt) && is.na(i_txt)) return(FALSE)
  if (!is.data.frame(i_txt)) stop(i_msgPart, " must be a data frame")
  if (nrow(i_txt) <= 0L) return(FALSE)

  # text frame must have $name, $lang, $descr, $note column
  if (is.null(i_txt$"name") || is.null(i_txt$"lang") ||
      is.null(i_txt$"descr") || is.null(i_txt$"note")) {
    stop(i_msgPart, " must have $name, $lang, $descr, $note columns")
  }

  # language code must NOT NULL and in the lang_lst table
  if (any(is.na(i_txt$"lang"))) {
    stop(i_msgPart, " must have $lang NOT NULL")
  }
  if (!all(i_txt$"lang" %in% i_langRs$lang_code)) {
    stop("invalid (or empty) language of ", i_msgPart)
  }

  # description must NOT NULL
  if (any(is.na(i_txt$"descr"))) {
    stop(i_msgPart, " must have $descr NOT NULL")
  }

  return(TRUE)  # valid and not empty
}
