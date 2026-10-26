#' @noRd
find_msvc_toolset <- function() {
  # Construct vswhere path
  vswhere <- file.path(
    Sys.getenv("ProgramFiles(x86)"),
    "Microsoft Visual Studio",
    "Installer",
    "vswhere.exe"
  )

  # Get Visual Studio meta
  vs_meta <- system2(
    command = vswhere,
    args = c(
      "-latest",
      "-format",
      "json"
    ),
    stdout = TRUE
  )

  # Abort if no meta found
  if (! length(vs_meta)) {
    cli::cli_abort("No Visual Studio meta data could be found on your machine.")
  }

  # Get install version
  install_v <- jsonlite::fromJSON(vs_meta) |>
    dplyr::pull(installationVersion) |>
    stringr::str_extract("^[0-9]+") |>
    as.integer()

  # Construct lookup table
  lookup <- dplyr::tibble(
    install_v = 16:18,
    toolset_v = paste0("v", c(142:143,145))
  )

  # Exit
  lookup$toolset_v[lookup$install_v == install_v]
}
