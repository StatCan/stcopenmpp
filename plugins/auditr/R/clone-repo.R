#' @noRd
clone_repo <- function(git_url, ...) {
  # Get args
  args <- list(...)

  # Validate args
  if (missing(git_url)) {
    cli::cli_abort("Please set the {.var git_url} argument.")
  }

  # Render console message
  cli::cli_alert_info("Loading model {.url {git_url}}. Hang tight ...")
  if (! is.null(args[["git_commit"]])) {
    cli::cli_alert_info("Checking out model repo at {.var {args$git_commit}}.")
  }

  # Set arguments if unset
  if (is.null(args[["dest"]])) {
    args$dest <- paste0(
      getwd(),
      "/auditr-downloads/",
      tools::file_path_sans_ext(basename(git_url))
    )
  }

  # Delete args$dest if exists
  unlink(
    x = args$dest,
    recursive = TRUE,
    force = TRUE
  )

  # Clone repo
  output <- suppressWarnings(
    system2(
      command = Sys.which("git"),
      args = c(
        "clone",
        paste0(
          "https://",
          ifelse(
            test = ! length(
              c(is.null(args[["git_user"]]), is.null(args["git_token"]))
            ),
            yes = paste0(
              args$git_user,
              ":",
              args$git_token,
              "@",
            ),
            no = ""
          ),
          gsub(pattern = "https://", replacement = "", x = git_url)
        ),
        args$dest
      ),
      stdout = TRUE,
      stderr = TRUE
    )
  )

  # Render output to console
  cli::cli_text("")
  cli::cli_alert_info(output)

  # Checkout commit if set
  if(! is.null(args[["git_commit"]])) {
    output <- checkout_repo(
      dest = args$dest,
      commit = args$git_commit,
      .quiet = TRUE
    )
  } else {
    # Get last commit
    output <- withr::with_dir(
      new = args$dest,
      code = args$git_commit <- system2(
        command = Sys.which("git"),
        args = c(
          "log",
          "-1",
          "--format=%H"
        ),
        stdout = TRUE,
        stderr = TRUE
      )
    )
  }

  # Render console message
  cli::cli_text("")
  cli::cli_alert_success("Done!")
  cli::cli_text("")

  # Exit
  list(dest = args$dest, commit = args$git_commit)
}
