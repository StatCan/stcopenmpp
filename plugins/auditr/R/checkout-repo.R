#' @noRd
checkout_repo <- function(dest, commit, .quiet = FALSE) {
  # Validate args
  if (missing(dest)) cli::cli_abort("Please set the {.var dest} argument.")
  if (missing(commit)) cli::cli_abort("Please set the {.var commit} argument.")

  # Render console message
  if (isFALSE(.quiet)) {
    cli::cli_alert_info(
      paste(
        "Checking out model repo {.path {dest}}",
        "at {.var {commit}}. Hang tight ..."
      )
    )
  }

  # Checkout commit
  output <- withr::with_dir(
    new = dest,
    code = system2(
      command = Sys.which("git"),
      args = c(
        "checkout",
        commit
      ),
      stdout = TRUE,
      stderr = TRUE
    )
  )

  # Render console message
  if (isFALSE(.quiet)) {
    cli::cli_text("")
    cli::cli_alert_success("Done!")
    cli::cli_text("")
  }

  # Exit
  list(dest = dest, commit = commit)
}
