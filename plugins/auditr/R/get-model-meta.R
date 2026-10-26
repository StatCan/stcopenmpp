#' @noRd
get_model_meta <- function(path, sln) {
  # Validate args
  if (missing(path)) cli::cli_abort("Please set the {.var path} argument.")
  if (missing(sln)) cli::cli_abort("Please set the {.var sln} argument.")

  # Find full path to sln
  sln_path <- list.files(
    path = path,
    pattern = sln,
    full.names = TRUE,
    recursive = TRUE,
    ignore.case = FALSE
  )

  # If path found
  if (length(sln_path)) {
    model_path <- dirname(sln_path)
    model <- basename(model_path)
  } else {
    model_path <- model <- NA
    cli::cli_abort("{.file {sln}} could not be found in {.url {path}}.")
  }

  # Exit
  list(model = model, model_path = model_path)
}
