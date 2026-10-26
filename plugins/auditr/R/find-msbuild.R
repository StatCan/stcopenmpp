#' @noRd
find_msbuild <- function() {
  # Construct vswhere path
  vswhere <- file.path(
    Sys.getenv("ProgramFiles(x86)"),
    "Microsoft Visual Studio",
    "Installer",
    "vswhere.exe"
  )

  # Find msbuild path
  msbuild <- system2(
    command = vswhere,
    args = c(
      "-latest",
      "-requires", "Microsoft.Component.MSBuild",
      "-find", "MSBuild\\**\\Bin\\MSBuild.exe"
    ),
    stdout = TRUE
  )

  # Abort if msbuild not found
  if (! length(msbuild)) {
    cli::cli_abort(
      "{.var {.strong MSBuild.exe}} could not be found on your machine."
    )
  }

  # Exit
  msbuild
}
