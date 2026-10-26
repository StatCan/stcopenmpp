#' @noRd
build_model <- function(
  om_root,
  model_path,
  mode = "release",
  platform = "x64",
  ...
) {
  # Render console message
  cli::cli_h2(
    paste0(
      cli::make_ansi_style("#af3c43")("\U1F341"),
      cli::make_ansi_style("#3465a4")("{.fn build_model} function")
    )
  )

  # Get optional args
  args <- list(...)

  # Check if windows environment
  is_windows_env <- .Platform$OS.type == "windows"

  # Validate args
  if (missing(om_root)) {
    cli::cli_abort("Please set the {.var om_root} argument.")
  }
  if (missing(model_path)) {
    cli::cli_abort("Please set the {.var model_path} argument.")
  }
  if (is_windows_env && is.null(args[["sln"]])) {
    cli::cli_abort("Please set the {.var sln} argument.")
  }

  # If Windows environment
  if (is_windows_env) {
    # Find msbuild and platform_toolset
    msbuild <- find_msbuild()
    platform_toolset <- find_msvc_toolset()

    # Explicitly set platform toolset in model project file
    update_platform_toolset <- sapply(
      X = model_path,
      FUN = function(x) {
        # Download project file
        vcxproj <- readLines(
          con = file.path(x, "/ompp/Model.vcxproj"),
          warn = FALSE
        )

        # Update project file
        vcxproj <- gsub(
          pattern = "<PlatformToolset>.*?</PlatformToolset>",
          replacement = paste0(
            "<PlatformToolset>",
            platform_toolset,
            "</PlatformToolset>"
          ),
          x = vcxproj
        )

        # Save project file
        save <- writeLines(
          text = vcxproj,
          con = file.path(x, "/ompp/Model.vcxproj")
        )
      }
    )
  }

  # Construct jobs
  jobs <- dplyr::tibble(om_root, model_path, sln, mode, platform)

  # Iterate jobs
  builds <- lapply(
    X = 1:nrow(jobs),
    FUN = function(x) {
      # If windows environment
      if (is_windows_env) {
        # Remove previous build
        unlink(
          x = paste0(jobs$model_path, "/ompp/", c("bin", "build", "src")),
          recursive = TRUE,
          force = TRUE
        )

        # Initialize model build process
        p <- processx::process$new(
          command = msbuild,
          args = c(
            jobs$sln[x],
            "/t:Rebuild",
            paste0("/p:Configuration=", jobs$mode[x]),
            paste0("/p:OM_ROOT=", jobs$om_root[x]),
            paste0("/p:Platform=", jobs$platform[x]),
            paste0("/p:PlatformToolset=", platform_toolset),
            "/p:OPEN_MODEL_WEB_UI=false"
          ),
          wd = jobs$model_path[x],
          stdout = file.path(jobs$model_path[x], "build.log"),
          stderr = file.path(jobs$model_path[x], "error.log")
        )
      } else {
        # Remove previous build
        unlink(
          x = paste0(jobs$model_path, "/ompp-linux/", c("bin", "build", "src")),
          recursive = TRUE,
          force = TRUE
        )

        # Initialize model build process
        p <- processx::process$new(
          command = "make",
          args = c(
            paste0(
              "RELEASE=",
              ifelse(test = mode == "release", yes = 1, no = 0)
            ),
            "all",
            "publish"
          ),
          wd = jobs$model_path[x],
          stdout = file.path(jobs$model_path[x], "build.log"),
          stderr = file.path(jobs$model_path[x], "error.log")
        )
      }
    }
  )

  # Get build number
  build_n <- length(builds)

  # Render console message
  cli::cli_alert_info("Building {build_n} model{?s}. Hang tight ...")
  cli::cli_text("")

  # While model build(s) underway, wait here
  while (sum(sapply(X = builds, FUN = function(x) x$is_alive()))) Sys.sleep(1)

  # Iterate model build(s) and check exit status
  errors <- sapply(X = builds, FUN = function(x) x$get_exit_status() != 0)

  # Get error number
  error_n <- sum(errors)

  # Render console message and exit
  if (! error_n) {
    cli::cli_alert_success("Done!")
    return(TRUE)
  } else {
    cli::cli_alert_danger("Build unsuccessful for the following models:")
    cli::cli_text("")
    for(path in model_path[which(errors > 0)]) cli::cli_alert("{.file {path}}")
    return(FALSE)
  }
}
