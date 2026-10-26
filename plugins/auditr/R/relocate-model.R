#' @noRd
relocate_model <- function(from, to) {
  # Validate args
  if (missing(from)) cli::cli_abort("Please set the {.var from} argument.")
  if (missing(to)) cli::cli_abort("Please set the {.var to} argument.")

  # Remove to path if exists
  unlink(
    x = to,
    recursive = TRUE,
    force = TRUE
  )

  # Move from path to to path
  output <- file.rename(
    from = from,
    to = to
  )

  # Exit
  if (output) to else FALSE
}
