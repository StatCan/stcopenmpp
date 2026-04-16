#' @title R6 class: `auditr`
#' @description `auditr` is an R6 class that provides methods for auditing two versions of stcopenmpp.
#' @export

auditr <- R6::R6Class(
  classname = "auditr",
  public = list(
    #' @field args A list of arguments passed into `auditr$new()`. See documentation for `auditr$new()` for details.
    args = list(),

    #' @field code A list of vectors representing code chunks. Any time an `auditr` method is called, the code chunk is saved and rendered in the audit report when `$load_report()` is called.
    code = list(),

    #' @field results A list of four tibbles: `audit`, `output_tables`, `model_runs` and `performance`.
    results = list(),

    #' @description This method creates an instance of an `auditr` class.
    #' @param releases Required: a length-two character vector representing the URLs or full paths to two releases of `stcopenmpp`.
    #' @param labels Required: a length-two character vector representing the labels for the two releases (default: `paste0("Release ", 1:2)`).
    #' @param sln Required: a length-one character vector representing the model solution file name.
    #' @param mode Required: a length-one character vector representing the model mode to use during the model build (`release` or `debug`; default: `release`).
    #' @param bit Required: a length-one numeric vector representing the the model architecture to use during the model build (`64` or `32`; default: `64`).
    #' @param git_url Required: a length-one character vector representing the URL to the model repo.
    #' @param git_user Optional: a length-one character vector representing the repo username.
    #' @param git_token Optional: a length-one character vector representing the repo personal access token.
    #' @param git_commit Optional: a length-one character vector representing commit or tag to checkout before building the model.
    #' @param keep_vcxproj Optional: a length-one logical vector representing whether to keep the model's `Model.vcxproj` file when loaded or to discard and download a fresh template from the RiskPaths model (default: `TRUE`).
    #' @param dir Optional: a length-one character vector representing the full path to where releases, models, model run meta and reports will be stored (default: `getwd()`).
    #' @param db Optional: a length-one character vector representing the full path to where the model `.sqlite` file will be located for model runs.
    #' @param vs_cmd Optional: a length-one character vector representing the full path to the Microsoft Visual Studio terminal. This is required if performing an audit in a Windows environment.
    #' @param platform_toolset Optional: a length-one character vector representing the version of C++ build tools to use (e.g., `"v142"`, `"v143"`). This is only relevant for audits in Windows.
    #' @param include Optional: a character vector representing the full path(s) to additional C++ libraries to include.
    #' @param digits Optional: a length-one integer vector representing the number of digits to use when comparing output table values during audits.
    #' @return Returns the `auditr` object invisibly.

    initialize = function(
    releases,
    labels = paste0("Release ", 1:2),
    sln,
    mode = "release",
    bit = 64,
    git_url,
    git_user,
    git_token,
    git_commit,
    keep_vcxproj = TRUE,
    dir = getwd(),
    db,
    vs_cmd,
    platform_toolset,
    include,
    digits = 10
    ) {
      # Bind args
      self$args <- as.list(environment())
      self$args$self <- NULL
      self$args <- lapply(
        X = self$args,
        FUN = function(x) if(is.symbol(x)) NULL else x
      )

      # Format arguments
      if(! is.null(self$args$git_token)) {
        private$git_token <- self$args$git_token
        self$args$git_token <- sodium::password_store(self$args$git_token)
      }
      self$args$vs_cmd <- shQuote(self$args$vs_cmd)
      self$args$mode <- stringr::str_to_title(self$args$mode)
      if(self$args$bit == 64) self$args$bit <- "x64" else self$args$bit <- "Win32"

      # Save commands history
      utils::savehistory()

      # Load commands history
      self$code$new <- readLines(".Rhistory")

      # If git_token set, censor anything between single or double quotes
      git_token <- suppressWarnings(max(which(stringr::str_detect(string = self$code$new, pattern = "git_token"))))
      if(length(git_token)) {
        self$code$new[git_token] <- gsub(
          pattern = "(['\"])(.*?)\\1",
          replacement = "\\1********************\\1",
          x = self$code$new[git_token]
        )
      }

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$new, pattern = "auditr[$]new")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$new[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$new <- styler::style_text(
        self$code$new[element_n:length(self$code$new)]
      )

      # Cache code
      private$cache_code()

      # Set release labels and paths
      self$args$release1_label = self$args$labels[1]
      self$args$release2_label = self$args$labels[2]
      self$args$release1 <- self$args$releases[1]
      self$args$release2 <- self$args$releases[2]
      self$args$release1_path = paste0(
        self$args$dir,
        "/auditr-downloads/",
        private$strip_file_ext(basename(self$args$release1))
      )

      self$args$release2_path = paste0(
        self$args$dir,
        "/auditr-downloads/",
        private$strip_file_ext(basename(self$args$release2))
      )

      # Get repo path
      self$args$repo_path = paste0(
        self$args$dir,
        "/auditr-downloads/",
        private$strip_file_ext(self$args$git_url)
      )

      # Get model name
      self$args$model <- basename(self$args$repo_path)

      # If db set, create paths and ensure folders exist
      if(! is.null(self$args$db)) {
        timestamp <- gsub(" |[:]|[.]", "-", Sys.time())
        self$args$release1_db <- paste0(self$args$db, "/", timestamp, "/", basename(self$args$release1_path), "/", self$args$model, ".sqlite")
        self$args$release2_db <- paste0(self$args$db, "/", timestamp, "/", basename(self$args$release2_path), "/", self$args$model, ".sqlite")
        dir.create(
          path = paste0(self$args$db, "/", timestamp, "/", basename(self$args$release1_path)),
          showWarnings = FALSE,
          recursive = TRUE
        )
        dir.create(
          path = paste0(self$args$db, "/", timestamp, "/", basename(self$args$release2_path)),
          showWarnings = FALSE,
          recursive = TRUE
        )
      }

      # Get OS and OS-related args
      self$args$os <- .Platform$OS.type
      if(tolower(self$args$os) == "windows") {
        self$args$os_details <- paste0(
          stringr::str_to_title(self$args$os),
          " (",
          Sys.info()["release"],
          " ",
          Sys.info()["version"],
          ")"
        )
      } else {
        self$args$os_details <- paste0(
          ifelse(
            test = stringr::str_detect(string = tolower(self$args$release1), pattern = "ubuntu"),
            yes = "Ubuntu",
            no = "Debian"
          ),
          " (",
          Sys.info()["release"],
          " ",
          Sys.info()["version"],
          ")"
        )
      }

      if(self$args$os == "windows") {
        self$args$shell <- "cmd"
        self$args$shell_flag <- "/c"
        self$args$model_exe <- paste0(self$args$model, ".exe")
        self$args$dbcopy_exe <- "dbcopy.exe"
        self$args$dbget_exe <- "dbget.exe"
        self$args$ompp_dir <- "/ompp/"
      } else {
        self$args$shell <- "sh"
        self$args$shell_flag <- "-c"
        self$args$model_exe <- paste0("./", self$args$model)
        self$args$dbcopy_exe <- "dbcopy"
        self$args$dbget_exe <- "dbget"
        self$args$ompp_dir <- "/ompp-linux/"
      }

      # Set more paths
      self$args$release1_model_path <- paste0(self$args$release1_path, "/models/", self$args$model)
      self$args$release1_model_bin_path <- paste0(self$args$release1_model_path, self$args$ompp_dir, "bin")
      self$args$release2_model_path <- paste0(self$args$release2_path, "/models/", self$args$model)
      self$args$release2_model_bin_path <- paste0(self$args$release2_model_path, self$args$ompp_dir, "bin")

      # Sort args
      self$args <- self$args[order(names(self$args))]

      # Exit
      return(invisible(self))
    },

    #' @description This method downloads and extracts `stcopenmpp` archived releases (e.g., `.zip`, `.tar.gz`, `.tar.xz`).
    #' @return Returns the `auditr` object invisibly.

    load_releases = function() {
      # Save command history
      utils::savehistory()

      # Load command history
      self$code$load_releases <- readLines(".Rhistory")

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$load_releases, pattern = "[$]load_releases")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$load_releases[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$load_releases <- styler::style_text(
        self$code$load_releases[element_n:length(self$code$load_releases)]
      )

      # Cache code
      private$cache_code()

      # Render console message
      cli::cli_h2(paste0(cli::make_ansi_style("#af3c43")("\U1F341"), cli::make_ansi_style("#3465a4")("{.fn $load_releases} method")))

      # Render console message
      cli::cli_alert_info("Loading releases. Hang tight ...")
      cli::cli_text("")

      # Clean up
      unlink(
        x = list.dirs(path = paste0(self$args$dir, "/auditr-downloads"), recursive = FALSE),
        recursive = TRUE,
        force = TRUE
      )

      # Download/extract releases
      private$load_release(url = self$args$release1, dest = self$args$release1_path)
      private$load_release(url = self$args$release2, dest = self$args$release2_path)

      # Set paths
      self$args$dbcopy_path <- list.files(path = paste0(self$args$release1_path, "/bin"), pattern = paste0(self$args$dbcopy_exe, "$"), full.names = TRUE)
      self$args$dbget_path <- list.files(path = paste0(self$args$release1_path, "/bin"), pattern = paste0(self$args$dbget_exe, "$"), full.names = TRUE)

      # Save args
      saveRDS(
        object = self$args,
        file = paste0(self$args$dir, "/auditr-downloads/args.rds")
      )

      # Render console message
      cli::cli_alert_success("Done!")

      # Exit
      return(invisible(self))
    },

    #' @description This method downloads a model from a repo via `git clone`.
    #' @return Returns the `auditr` object invisibly.

    load_model = function() {
      # Save command history
      utils::savehistory()

      # Load command history
      self$code$load_model <- readLines(".Rhistory")

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$load_model, pattern = "[$]load_model")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$load_model[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$load_model <- styler::style_text(
        self$code$load_model[element_n:length(self$code$load_model)]
      )

      # Cache code
      private$cache_code()

      # Render console message
      cli::cli_h2(paste0(cli::make_ansi_style("#af3c43")("\U1F341"), cli::make_ansi_style("#3465a4")("{.fn $load_model} method")))

      # Render console message
      cli::cli_alert_info("Loading model. Hang tight ...")

      # Remove destination path if it exists
      unlink(
        x = self$args$repo_path,
        recursive = TRUE,
        force = TRUE
      )

      # Clone repo
      if(sum(sapply(X = self$args[c("git_user", "git_url", "git_token")], FUN = function(x) is.null(x))) == 0) {
        git_clone <- suppressWarnings(
          system2(
            command = Sys.which("git"),
            args = c(
              "clone",
              paste0(
                "https://",
                self$args$repo_user,
                ":",
                private$git_token,
                "@",
                gsub(pattern = "https://", replacement = "", x = self$args$git_url)
              ),
              paste0(self$args$dir, "/auditr-downloads/", self$args$model)
            ),
            stdout = TRUE,
            stderr = TRUE
          )
        )
      } else {
        git_clone <- suppressWarnings(
          system2(
            command = Sys.which("git"),
            args = c(
              "clone",
              self$args$git_url,
              paste0(self$args$dir, "/auditr-downloads/", self$args$model)
            ),
            stdout = TRUE,
            stderr = TRUE
          )
        )
      }

      # Render console message
      cli::cli_text("")
      cli::cli_alert(git_clone)

      # Switch to commit if set
      if(! is.null(self$args$git_commit)) {
        git_checkout <- suppressWarnings(
          system2(
            command = Sys.which("git"),
            args = c(
              "-C",
              self$args$repo_path,
              "checkout",
              self$args$git_commit
            ),
            stdout = TRUE,
            stderr = TRUE
          )
        )

        # Render console message
        cli::cli_text("")
        cli::cli_alert(git_checkout)
      }

      # Get last commit
      withr::with_dir(
        new = paste0(self$args$dir, "/auditr-downloads/", self$args$model),
        code = {
          self$args$git_commit_hash <- system2(
            command = Sys.which("git"),
            args = c(
              "log",
              "-1",
              "--format=%H"
            ),
            stdout = TRUE,
            stderr = TRUE
          )

          # Re-sort args
          self$args <- self$args[order(names(self$args))]
        }
      )

      # If self$args$sln not in repo root
      if(! file.exists(paste0(self$args$repo_path, "/", self$args$sln))) {
        # Look for correct path
        sln <- list.files(
          path = self$args$repo_path,
          pattern = self$args$sln,
          full.names = TRUE,
          recursive = TRUE
        )

        # If correct path found
        if(length(sln)) {
          # Store old model
          old_model <- self$args$model

          # Update self$args
          self$args$repo_path <- dirname(sln)
          self$args$model <- basename(dirname(sln))
          model_paths <- c(
            "model_exe",
            "release1_model_bin_path",
            "release1_model_path",
            "release2_model_bin_path",
            "release2_model_path"
          )
          self$args[model_paths] <- gsub(
            pattern = old_model,
            replacement = self$args$model,
            x = c(
              self$args[model_paths]
            )
          )
        } else {
          # Render console message
          cli::cli_abort("The {.emph sln} argument, {.var {self$args$sln}}, could not be located in the repo.")
        }
      }

      # Remove repo from self$args$release1_path/models and self$args$release2_path/models if it exists
      unlink(
        x = paste0(c(self$args$release1_model_path, self$args$release2_model_path)),
        recursive = TRUE
      )

      # Copy repo to self$args$release1_path/models and self$args$release2_path/models
      suppressWarnings(
        file.copy(
          from = self$args$repo_path,
          to = paste0(self$args$release1_path, "/models/"),
          recursive = TRUE,
        )
      )

      suppressWarnings(
        file.copy(
          from = self$args$repo_path,
          to = paste0(self$args$release2_path, "/models/"),
          recursive = TRUE
        )
      )

      # Save args
      saveRDS(
        object = self$args,
        file = paste0(self$args$dir, "/auditr-downloads/args.rds")
      )

      # Render console message
      cli::cli_text("")
      cli::cli_alert_success("Done!")
      cli::cli_text("")

      # Exit
      return(invisible(self))
    },

    #' @description This method compiles a model in both `stcopenmpp` releases asynchronously.
    #' @return Returns the `auditr` object invisibly.

    build_models = function() {
      # Save command history
      utils::savehistory()

      # Load command history
      self$code$build_models <- readLines(".Rhistory")

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$build_models, pattern = "[$]build_models")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$build_models[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$build_models <- styler::style_text(
        self$code$build_models[element_n:length(self$code$build_models)]
      )

      # Cache code
      private$cache_code()

      # Update self$args
      if(file.exists(paste0(self$args$dir, "/auditr-downloads/args.rds"))) {
        args <- readRDS(paste0(self$args$dir, "/auditr-downloads/args.rds"))

        if(self$args$git_url == args$git_url & self$args$model != args$model) {
          arg_labels <- c(
            "model",
            "model_exe",
            "repo_path",
            paste0("release", 1:2, "_model_bin_path"),
            paste0("release", 1:2, "_model_path"),
            paste0(c("dbcopy", "dbget"), "_path")
          )
          self$args[arg_labels] <- args[arg_labels]

          # Sort args
          self$args <- self$args[order(names(self$args))]
        }
      }

      # Render console message
      cli::cli_h2(paste0(cli::make_ansi_style("#af3c43")("\U1F341"), cli::make_ansi_style("#3465a4")("{.fn $build_models} method")))
      cli::cli_alert_info("Model builds underway in parallel. Hang tight ...")
      cli::cli_text("")
      cli::cli_progress_bar("Compiling ...")

      # Build models asynchronously
      private$build_model(om_root = self$args$release1_path, model_path = self$args$release1_model_path, model_bin_path = self$args$release1_model_bin_path)
      private$build_model(om_root = self$args$release2_path, model_path = self$args$release2_model_path, model_bin_path = self$args$release2_model_bin_path)

      # Monitor progress
      while (TRUE) {
        if (sum(! file.exists(paste0(c(self$args$release1_model_bin_path, self$args$release2_model_bin_path), "/build.done"))) == 0) break
        Sys.sleep(1)
        cli::cli_progress_update()
      }

      # Stop progress bar
      cli::cli_progress_done()

      # Import build logs
      build_logs <- dplyr::bind_rows(
        sapply(
          X = paste0(c(self$args$release1_model_bin_path, self$args$release2_model_bin_path), "/build.log"),
          FUN = function(x) {
            readLines(con = x, warn = FALSE) |>
              dplyr::as_tibble() |>
              dplyr::mutate(path = x)
          },
          simplify = FALSE
        )
      )

      # Validate builds
      if(sum(stringr::str_detect(tolower(build_logs$value), "build succeeded.")) == 2) {
        # Copy databases
        file.copy(
          from = paste0(c(self$args$release1_model_bin_path, self$args$release2_model_bin_path), "/", self$args$model, ".sqlite"),
          to = paste0(c(self$args$release1_model_bin_path, self$args$release2_model_bin_path), "/", self$args$model, "-copy.sqlite")
        )

        # Render console message
        cli::cli_alert_success("Done!")
      } else {
        if(sum(stringr::str_detect(string = tolower(build_logs$value), pattern = "unresolved external symbol"))) {
          # Render console message
          cli::cli_abort("Model builds unsuccessful due to {.emph unresolved external symbol} errors. Please call {.fn $rebuild_openm} and then retry {.fn $build_models}.")
        } else {
          # Render console message
          cli::cli_abort("Model builds unsuccessful. Please try again.")
        }
      }

      # Exit
      return(invisible(self))
    },

    #' @description This method re-compiles the OpenM++ library in both `stcopenmpp` releases asynchronously.
    #' @return Returns the `auditr` object invisibly.

    rebuild_openm = function() {
      # Save command history
      utils::savehistory()

      # Load command history
      self$code$rebuild_openm <- readLines(".Rhistory")

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$rebuild_openm, pattern = "[$]rebuild_openm")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$rebuild_openm[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$rebuild_openm <- styler::style_text(
        self$code$rebuild_openm[element_n:length(self$code$rebuild_openm)]
      )

      # Cache code
      private$cache_code()

      # Render console message
      cli::cli_h2(paste0(cli::make_ansi_style("#af3c43")("\U1F341"), cli::make_ansi_style("#3465a4")("{.fn $rebuild_openm} method")))
      cli::cli_alert_info("Rebuild of OpenM++ libraries underway in parallel. Hang tight ...")
      cli::cli_text("")
      cli::cli_progress_bar("Compiling ...")

      # Build models asynchronously
      private$.rebuild_openm(om_root = self$args$release1_path)
      private$.rebuild_openm(om_root = self$args$release2_path)

      # Monitor progress
      while (TRUE) {
        if (sum(! file.exists(paste0(c(paste0(self$args$release1_path, "/openm"), paste0(self$args$release2_path, "/openm")), "/build.done"))) == 0) break
        Sys.sleep(1)
        cli::cli_progress_update()
      }

      # Stop progress bar
      cli::cli_progress_done()

      # Import build logs
      build_logs <- dplyr::bind_rows(
        sapply(
          X = paste0(c(paste0(self$args$release1_path, "/openm"), paste0(self$args$release2_path, "/openm")), "/build.log"),
          FUN = function(x) {
            readLines(con = x, warn = FALSE) |>
              dplyr::as_tibble() |>
              dplyr::mutate(path = x)
          },
          simplify = FALSE
        )
      )

      # Validate builds
      if(sum(stringr::str_detect(tolower(build_logs$value), "build succeeded.")) == 2) {
        # Render console message
        cli::cli_alert_success("Done!")
      } else {
        # Render console message
        cli::cli_abort("Rebuilds of the OpenM libraries were unsuccessful. Please try again.")
      }

      # Exit
      return(invisible(self))
    },

    #' @description This method retrieves and stores a list of a model's non-hidden output tables via `dbcopy.exe`.
    #' @return Returns the `auditr` object invisibly.

    load_output_tables = function() {
      # Save command history
      utils::savehistory()

      # Load command history
      self$code$load_output_tables <- readLines(".Rhistory")

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$load_output_tables, pattern = "[$]load_output_tables")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$load_output_tables[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$load_output_tables <- styler::style_text(
        self$code$load_output_tables[element_n:length(self$code$load_output_tables)]
      )

      # Cache code
      private$cache_code()

      # Update self$args
      if(file.exists(paste0(self$args$dir, "/auditr-downloads/args.rds"))) {
        args <- readRDS(paste0(self$args$dir, "/auditr-downloads/args.rds"))

        if(self$args$git_url == args$git_url & self$args$model != args$model) {
          arg_labels <- c(
            "model",
            "model_exe",
            "repo_path",
            paste0("release", 1:2, "_model_bin_path"),
            paste0("release", 1:2, "_model_path"),
            paste0(c("dbcopy", "dbget"), "_path")
          )
          self$args[arg_labels] <- args[arg_labels]
        }

        if(is.null(self$args$git_commit)) self$args$git_commit <- args$git_commit
        if(is.null(self$args$git_commit_hash)) self$args$git_commit_hash <- args$git_commit_hash
      }

      # Render console message
      cli::cli_h2(paste0(cli::make_ansi_style("#af3c43")("\U1F341"), cli::make_ansi_style("#3465a4")("{.fn $load_output_tables} method")))

      # Abort if model_path doesn't exit
      if(! file.exists(paste0(self$args$release1_model_bin_path, "/", self$args$model_exe))) cli::cli_abort("Please run {.fn $build_models}.")

      # Render console message
      cli::cli_alert_info("Loading non-hidden output tables. Hang tight ...")

      # Download model meta
      withr::with_dir(
        new = self$args$release1_model_bin_path,
        code = {
          system2(
            command = self$args$dbcopy_path,
            args = c(
              "-m",
              self$args$model,
              "-OpenM.LogToConsole=false"
            )
          )
        }
      )

      # Connect to database
      connection <- DBI::dbConnect(
        RSQLite::SQLite(),
        dbname = paste0(self$args$release1_model_bin_path, "/", self$args$model, "-copy.sqlite")
      )

      # Import output table meta and bind to self$results$output_tables
      self$results$output_tables <- DBI::dbGetQuery(
        conn = connection,
        statement = sprintf(
          fmt = "
            SELECT
              l.table_hid AS id,
              l.table_name AS name,
              l2.descr AS label
            FROM
              table_dic l
            LEFT JOIN
              table_dic_txt l2
            ON
              l.table_hid = l2.table_hid
            LEFT JOIN
              model_table_dic r
            ON
              l.table_hid = r.table_hid
            WHERE
              l2.lang_id LIKE (SELECT MIN(lang_id) FROM table_dic_txt)
              AND
              r.is_hidden LIKE %d
          ",
          0
        )
      ) |>
        dplyr::as_tibble()

      # Disconnect
      DBI::dbDisconnect(connection)

      # Render console message
      cli::cli_text("")
      cli::cli_alert("Metadata for {nrow(self$results$output_tables)} output table{?s} stored in {.var $results$output_tables}.")
      cli::cli_text("")
      cli::cli_alert_success("Done!")

      # Exit
      return(invisible(self))
    },

    #' @description This method initiates model runs in both `stcopenmpp` releases asynchronously.
    #' @param parameters Optional: a named vector (default: `c()`).
    #' @param sub_from Optional: a named vector (default: `c()`).
    #' @param cases Required: a length-one numeric vector representing the number of cases to run (default: `1e6`).
    #' @param threads Required: a length-one numeric vector representing the number of threads to use during the model run (default: `1`).
    #' @param sub_samples Required: a length-one numeric vector representing the number of sub-samples to use during the model run (default: `1`).
    #' @param param_dir Optional: a length-one character vector representing the full path to the directory where the parameter files are located (default: `NULL`).
    #' @param tables_per_run Optional: a length-one numeric vector representing the number of output tables per model run (default: `25`).
    #' @return Returns the `auditr` object invisibly.

    run_models = function(
    parameters = c(),
    sub_from = c(),
    cases = 1e6,
    threads = 1,
    sub_samples = 1,
    param_dir = NULL,
    tables_per_run = 25
    ) {
      # Save command history
      utils::savehistory()

      # Load command history
      self$code$run_models <- readLines(".Rhistory")

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$run_models, pattern = "[$]run_models")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$run_models[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$run_models <- styler::style_text(
        self$code$run_models[element_n:length(self$code$run_models)]
      )

      # Cache code
      private$cache_code()

      # Get list of output tables if not already set
      if(! "output_tables" %in% names(self$results) | ! length(self$results$output_tables)) {
        # Render console message
        cli::cli_abort("Please call {.fn $load_output_tables} and then retry {.fn run_models}.")
      }

      # Clean up
      self$results <- list(output_tables = self$results$output_tables)

      # Render console message
      cli::cli_h2(paste0(cli::make_ansi_style("#af3c43")("\U1F341"), cli::make_ansi_style("#3465a4")("{.fn $run_models} method")))
      cli::cli_alert_info("Model runs underway in parallel. Hang tight ...")
      cli::cli_text("")

      # Create table groupings based on tables_per_run
      table_groupings <- split(
        x = self$results$output_tables$name,
        f = ceiling(seq_along(self$results$output_tables$name) / tables_per_run)
      )
      names(table_groupings) <- NULL

      # Iterate table_groupings
      for(i in 1:length(table_groupings)) {
        # Set run name
        run_name <- paste0("run-", stringr::str_pad(string = i, width = max(nchar(length(table_groupings)), 2), pad = "0"))

        # Set db
        if(! is.null(self$args$db)) db1 <- self$args$release1_db else db1 <- NULL
        if(! is.null(self$args$db)) db2 <- self$args$release2_db else db2 <- NULL

        # Initiate model runs asynchronously
        private$run_model(
          run_name = run_name,
          om_root = self$args$release1_path,
          model_bin_path = self$args$release1_model_bin_path,
          table_grouping = table_groupings[[i]],
          parameters,
          sub_from,
          cases,
          threads,
          sub_samples,
          param_dir,
          db = db1
        )
        private$run_model(
          run_name = run_name,
          om_root = self$args$release2_path,
          model_bin_path = self$args$release2_model_bin_path,
          table_grouping = table_groupings[[i]],
          parameters,
          sub_from,
          cases,
          threads,
          sub_samples,
          param_dir,
          db = db2
        )

        # Comma-separate output tables
        output_tables <- paste0(table_groupings[[i]], collapse = ", ")

        # Render console message
        cli::cli_alert("{.strong Iteration {i}/{length(table_groupings)}} underway with the following output tables retained: {.var {output_tables}}.")

        # Wait for log files to be created
        while(TRUE) {
          # Get log file names
          log_files <- c(
            paste0(self$args$release1_model_bin_path, "/", self$args$model, ".log"),
            paste0(self$args$release2_model_bin_path, "/", self$args$model, ".log")
          )

          # Check if they exist
          if(sum(file.exists(log_files)) == 2) break

          # Pause
          Sys.sleep(5)
        }

        # Monitor progress
        while (TRUE) {
          # Look for errors
          logs <- c(
            readLines(con = paste0(self$args$release1_model_bin_path, "/", self$args$model, ".log")),
            readLines(con = paste0(self$args$release2_model_bin_path, "/", self$args$model, ".log"))
          ) |> tolower()

          error_fail_sums <- sum(stringr::str_detect(string = logs, pattern = "error|failed"))
          fake_error_sums <- sum(stringr::str_detect(string = logs, pattern = "not an error"))

          if(error_fail_sums > 0 && error_fail_sums > fake_error_sums) {
            cli::cli_abort(
              paste0(
                "One or more model runs encountered an error. Please consult the run logs ({.file ",
                paste0(self$args$release1_model_bin_path, "/", self$args$model, ".log"),
                "}, {.file ",
                paste0(self$args$release2_model_bin_path, "/", self$args$model, ".log"),
                "})."
              )
            )
          }

          # Look for run.done flags
          if (sum(! file.exists(paste0(c(self$args$release1_model_bin_path, self$args$release2_model_bin_path), "/run.done"))) == 0) break

          # Pause
          Sys.sleep(5)
        }

        # Set paths
        om_roots <- c(self$args$release1_path, self$args$release2_path)
        model_paths <- c(self$args$release1_model_bin_path, self$args$release2_model_bin_path)

        # Render console message
        cli::cli_alert("Getting model run results and auditing output tables.")

        # Get db paths
        if(! is.null(self$args$db)) {
          db_paths <- self$args[paste0("release", 1:length(model_paths), "_db")]
        } else {
          db_paths <- paste0(self$args[paste0("release", 1:length(model_paths), "_model_bin_path")], "/", self$args$model, ".sqlite")
        }

        # Set run_ids
        run_ids <- c()

        # Iterate model_paths
        for(j in 1:length(model_paths)) {
          # Connect to database
          connection <- DBI::dbConnect(
            RSQLite::SQLite(),
            dbname = db_paths[j]
          )

          # Get model run meta
          model_run_meta <- DBI::dbGetQuery(
            conn = connection,
            statement = "SELECT * FROM run_lst"
          ) |>
            dplyr::arrange(dplyr::desc(run_id)) |>
            dplyr::slice(1) |>
            dplyr::as_tibble()

          # Disconnect from database
          DBI::dbDisconnect(connection)

          # Update run_ids
          run_ids <- c(run_ids, model_run_meta$run_id)

          # Bind to self$results$model_runs
          self$results$model_runs <- dplyr::bind_rows(
            self$results$model_runs,
            model_run_meta |>
              dplyr::arrange(dplyr::desc(update_dt)) |>
              dplyr::slice(1) |>
              dplyr::mutate(
                release = paste0("release", j),
                release_path = self$args[paste0("release", j, "_path")] |>
                  unlist() |>
                  unname(),
                cases = cases,
                output_tables = output_tables
              )
          )
        }

        # Iterate output tables and do audit
        self$results$audit <- dplyr::bind_rows(
          self$results$audit,
          lapply(
            X = table_groupings[[i]],
            FUN = function(x) {
              # Save output table
              output_table <- x

              # Download output tables
              output <- lapply(
                X = 1:length(model_paths),
                FUN = function(x) {
                  # Connect to database
                  connection <- DBI::dbConnect(RSQLite::SQLite(), dbname = db_paths[x])

                  # Get run table
                  run_table <- DBI::dbGetQuery(
                    conn = connection,
                    statement = sprintf(
                      fmt = "
                        SELECT
                          l.run_id,
                          l.table_hid,
                          l.base_run_id,
                          l.value_digest,
                          r.table_name,
                          r.db_expr_table
                        FROM
                          run_table l
                        LEFT JOIN
                          table_dic r
                        ON
                          l.table_hid = r.table_hid
                        WHERE
                          l.run_id LIKE %d
                        AND
                          r.table_name LIKE '%s'
                      ",
                      run_ids[x],
                      output_table
                    )
                  ) |>
                    dplyr::as_tibble()

                  # Get output table
                  output_table <- DBI::dbGetQuery(
                    conn = connection,
                    statement = sprintf(
                      fmt = "SELECT * FROM %s WHERE run_id LIKE %d",
                      run_table$db_expr_table,
                      run_table$base_run_id
                    )
                  ) |>
                    dplyr::select(-run_id) |>
                    dplyr::rename(expr_name = expr_id)

                  # Disconnect from database
                  DBI::dbDisconnect(connection)

                  # Exit
                  return(output_table)
                }
              )

              # Round expr_value
              output[[1]]$expr_value <- round(x = output[[1]]$expr_value, digits = self$args$digits)
              output[[2]]$expr_value <- round(x = output[[2]]$expr_value, digits = self$args$digits)

              # Get row location of any value disparities between model runs
              value_disparities <- which(
                output[[1]]$expr_value !=
                  output[[2]]$expr_value
              )

              # Compute summary stats
              dplyr::tibble(
                table = x,
                table_size = sapply(names(output[[1]])[1:(ncol(output[[1]]) - 1)], function(x) length(unique(output[[1]][[x]]))) |>
                  unname() |>
                  paste0(collapse = " x "),
                expression_count = length(unique(output[[1]]$expr_name)),
                dimension_count = ncol(output[[1]]) - 2,
                value_count = nrow(output[[1]]),
                diff_count = length(value_disparities),
                diff_percent = diff_count / value_count * 100,
                min_diff = ifelse(
                  test = diff_count > 0,
                  yes = min(
                    x = value_disparities,
                    na.rm = TRUE
                  ),
                  no = NA
                ),
                max_diff = ifelse(
                  test = diff_count > 0,
                  yes = max(
                    x = value_disparities,
                    na.rm = TRUE
                  ),
                  no = NA
                ),
                median_diff = ifelse(
                  test = diff_count > 0,
                  yes = median(
                    x = value_disparities,
                    na.rm = TRUE
                  ),
                  no = NA
                )
              ) |>
                dplyr::mutate(
                  dplyr::across(
                    .cols = value_count:diff_count,
                    .fns = ~ format(
                      x = .x,
                      big.mark = ","
                    )
                  ),
                  dplyr::across(
                    .cols = diff_percent:median_diff,
                    .fns = ~ format(
                      round(
                        x = .x,
                        digits = 2
                      ),
                      big.mark = ",",
                      nsmall = 2
                    )
                  ),
                  dplyr::across(
                    .cols = min_diff:median_diff,
                    .fns = ~ ifelse(
                      test = .x == "NA",
                      yes = "-",
                      no = .x
                    )
                  )
                )
            }
          )
        )

        # Render console message
        cli::cli_alert("Cleaning up.")
        cli::cli_text("")

        # Clean up
        unlink(x = paste0(model_paths, "/", self$args$model, ".sqlite"))
      }

      # Render console message
      cli::cli_alert_info("Model builds done. Wrapping things up. Hang tight ...")
      cli::cli_text("")

      # Update and reshape $results$model_runs
      self$results$model_runs <- self$results$model_runs |>
        dplyr::mutate(
          run_number = dplyr::row_number(),
          dplyr::across(.cols = dplyr::everything(), .fns = ~ as.character(.x))
        ) |>
        tidyr::pivot_longer(
          cols = -c(run_name, release),
          names_to = "key"
        ) |>
        dplyr::mutate(
          key = key |>
            factor(
              levels = c(
                "run_number",
                "run_id",
                "model_id",
                "run_name",
                "run_stamp",
                "run_digest",
                "create_dt",
                "update_dt",
                "cases",
                "sub_count",
                "sub_started",
                "sub_restart",
                "sub_completed",
                "status",
                "value_digest",
                "release_path",
                "output_tables",
                "descr",
                "lang_code"

              )
            )
        ) |>
        dplyr::group_by(run_name, release, key) |>
        dplyr::reframe(value = dplyr::first(value)) |>
        tidyr::pivot_wider(
          names_from = release,
          values_from = value
        ) |>
        dplyr::select(-run_name) |>
        dplyr::rename(
          Variable = key,
          !! self$args$release1_label := release1,
          !! self$args$release2_label := release2
        )

      # Compute model run time
      self$results$performance <- self$results$model_runs |>
        dplyr::filter(Variable %in% paste0(c("create_", "update_"), "dt")) |>
        tidyr::pivot_longer(cols = -Variable) |>
        dplyr::mutate(value = as.POSIXct(value)) |>
        dplyr::reframe(
          run_start = min(value),
          run_end = max(value),
          run_time = difftime(time1 = run_end, time2 = run_start, units = "mins")
        )

      # Store auditr object
      dir.create(path = paste0(self$args$dir, "/auditr-model-runs"), showWarnings = FALSE, recursive = TRUE)
      rds_path <- paste0(self$args$dir, "/auditr-model-runs/", self$args$model, "-", gsub(" |[:]", "-", Sys.time()), ".rds")
      saveRDS(
        object = self,
        file = rds_path,
        compress = "xz"
      )

      # Re-sort $results elements
      self$results <- self$results[order(names(self$results))]

      # Render console message
      cli::cli_alert("Output table audit stored in {.var $results$audit}.")
      cli::cli_alert("Model run meta stored in {.var $results$model_runs}.")
      cli::cli_alert("{.var auditr} object stored in {.file {rds_path}}.")
      cli::cli_text("")
      cli::cli_alert_success("Done!")

      # Exit
      return(invisible(self))
    },

    #' @description This method renders an audit report in .html format.
    #' @param name Required: a length-one character vector representing the file name of the report (default: `paste0("auditr-report-", self$args$model)`).
    #' @param dir Required: a length-one character vector representing the full path to the location where the report will be exported in `.html` format (default: `paste0(self$args$dir, "/auditr-reports")`).
    #' @return Returns an .html-formatted report.

    load_report = function(name = paste0("auditr-report-", self$args$model), dir = paste0(self$args$dir, "/auditr-reports")) {
      # Save command history
      utils::savehistory()

      # Load command history
      self$code$load_report <- readLines(".Rhistory")

      # Set element number to start from
      element_n <- max(which(stringr::str_detect(string = self$code$load_report, pattern = "[$]load_report")))

      # Adjust if preceded by comments
      if(element_n > 1) {
        for(i in (element_n - 1):1) {
          if(grepl("^#", trimws(self$code$load_report[i]))) element_n <- i else break
        }
      }

      # Style code
      self$code$load_report <- styler::style_text(
        self$code$load_report[element_n:length(self$code$load_report)]
      )

      # Cache code
      private$cache_code()

      # Render console message
      cli::cli_h2(paste0(cli::make_ansi_style("#af3c43")("\U1F341"), cli::make_ansi_style("#3465a4")("{.fn $load_report} method")))
      cli::cli_alert_info("Rendering report in .html format. Hang tight ...")
      cli::cli_text("")

      # Throw error if suggested packages are not installed but needed
      if(! rlang::is_installed("kableExtra")) {
        cli::cli_abort("Please install the {.pkg kableExtra} package.")
      }

      # Create auditr-reports folder
      dir.create(
        path = paste0(self$args$dir, "/auditr-reports"),
        showWarnings = FALSE,
        recursive = FALSE
      )

      # Set report name
      timestamp <- gsub(" |[:]|[.]", "-", Sys.time())
      report_name_qmd <- paste0(name, "-", timestamp, ".qmd")
      report_name_html <- paste0(name, "-", timestamp, ".html")

      # Copy quarto doc to working directory (for now)
      copy_file <- file.copy(
        from = system.file("qmd", "auditr-report.qmd", package = "auditr"),
        to = paste0(getwd(), "/", report_name_qmd),
        overwrite = TRUE
      )

      # Render Quarto doc
      quarto::quarto_render(
        input = paste0(
          getwd(),
          "/",
          report_name_qmd
        ),
        execute_params = list(
          args = self$args,
          code = readRDS(file = paste0(self$args$dir, "/auditr-downloads/code.rds")),
          results = self$results
        )
      )

      # Remove qmd file
      remove_file <- file.remove(
        paste0(
          getwd(),
          "/",
          report_name_qmd
        )
      )

      # Copy quarto doc to dir
      if(dir != getwd()) {
        # Copy file
        copy_file <- file.copy(
          from = paste0(getwd(), "/", report_name_html),
          to = paste0(dir, "/", report_name_html),
          overwrite = TRUE
        )

        # Remove file from working directory
        remove_file <- file.remove(
          paste0(
            getwd(),
            "/",
            report_name_html
          )
        )
      }

      # Render console message
      cli::cli_text("The audit report is available here: {.url {paste0(dir, '/', report_name_html)}}")
      cli::cli_text("")
      cli::cli_alert_success("Done!")

      # Send report to browser
      utils::browseURL(url = paste0(dir, '/', report_name_html))

      # Exit
      return(invisible(self))
    }
  ),

  private = list(
    blue = function(x) cli::make_ansi_style("#3465a4")(x),

    build_model = function(om_root, model_path, model_bin_path) {
      # Set paths for downstream use
      sln_path <- paste0(model_path, '/', self$args$sln)

      if(isFALSE(self$args$keep_vcxproj)) {
        # Download Model.vcxproj file (default settings from RiskPaths model)
        vcxproj_file <- readLines(con = system.file("vcxproj", "Model.vcxproj", package = "auditr"))

        # Update platform toolset is set
        if(! is.null(self$args$platform_toolset)) {
          vcxproj_file <- gsub(
            pattern = "<PlatformToolset>.*?</PlatformToolset>",
            replacement = paste0("<PlatformToolset>", self$args$platform_tools, "</PlatformToolset>"),
            x = vcxproj_file
          )
        }

        # Update Model.vcxproj file
        update_model_vcxproj <- writeLines(
          text = vcxproj_file,
          con = list.files(
            path = paste0(model_path, "/ompp"),
            pattern = "model.vcxproj$",
            full.names = TRUE,
            ignore.case = TRUE
          )
        )
      }

      # Update platform toolset if set
      if(! is.null(self$args$platform_toolset)) {
        # Download Model.vcxproj file (default settings from RiskPaths model)
        vcxproj_file <- suppressWarnings(
          readLines(
            con = list.files(
              path = paste0(model_path, "/ompp"),
              pattern = "model.vcxproj$",
              full.names = TRUE,
              ignore.case = TRUE
            )
          )
        )

        vcxproj_file <- gsub(
          pattern = "<PlatformToolset>.*?</PlatformToolset>",
          replacement = paste0("<PlatformToolset>", self$args$platform_tools, "</PlatformToolset>"),
          x = vcxproj_file
        )

        # Update Model.vcxproj file
        update_model_vcxproj <- writeLines(
          text = vcxproj_file,
          con = list.files(
            path = paste0(model_path, "/ompp"),
            pattern = "model.vcxproj$",
            full.names = TRUE,
            ignore.case = TRUE
          )
        )
      }

      # Update additional includes if set
      if(! is.null(self$args$include)) {
        # Update OS-specific file
        if(self$args$os == "windows") {
          # Download Model.vcxproj file (default settings from RiskPaths model)
          vcxproj_file <- suppressWarnings(
            readLines(
              con = list.files(
                path = paste0(model_path, "/ompp"),
                pattern = "model.vcxproj$",
                full.names = TRUE,
                ignore.case = TRUE
              )
            )
          )

          # Remove any includes
          vcxproj_file <- gsub(
            pattern = "<ItemDefinitionGroup[\\s\\S]*?</ItemDefinitionGroup>",
            replacement = "",
            x = vcxproj_file
          )

          # Add includes
          includes <- paste0(
            "<ItemDefinitionGroup Condition=\"'$(Configuration)|$(Platform)'=='",
            self$args$mode,
            "|",
            self$args$bit,
            "'\">",
            "<ClCompile>"
          )

          for(i in 1:length(self$args$include)) {
            includes <- append(
              x = includes,
              values = paste0(
                "<AdditionalIncludeDirectories>",
                self$args$include[i],
                "</AdditionalIncludeDirectories>"
              )
            )
          }

          includes <- append(
            x = includes,
            values = "</ClCompile></ItemDefinitionGroup>"
          )

          # Find line with </project>
          project_line <- stringr::str_detect(
            string = tolower(vcxproj_file),
            pattern = "</project>"
          ) |> which()

          vcxproj_file <- c(
            vcxproj_file[1:(project_line - 1)],
            includes_windows,
            vcxproj_file[project_line]
          )

          # Update Model.vcxproj file
          update_model_vcxproj <- writeLines(
            text = vcxproj_file,
            con = list.files(
              path = paste0(model_path, "/ompp"),
              pattern = "model.vcxproj$",
              full.names = TRUE,
              ignore.case = TRUE
            )
          )
        } else {
          # Download makefile file
          makefile_file <- suppressWarnings(
            readLines(
              con = list.files(
                path = paste0(model_path),
                pattern = "makefile$",
                full.names = TRUE,
                ignore.case = TRUE
              )
            )
          )

          includes <- c()
          for(i in 1:length(self$args$include)) {
            includes <- append(
              x = includes,
              values = paste0("CXXFLAGS += -I", self$args$include[i])
            )
          }

          # Update makefile file
          update_makefile <- writeLines(
            text = c(
              makefile_file,
              includes
            ),
            con = list.files(
              path = paste0(model_path),
              pattern = "makefile$",
              full.names = TRUE,
              ignore.case = TRUE
            )
          )
        }
      }

      # Clean up
      unlink(
        x = paste0(model_path, self$args$ompp_dir, c("bin", "build", "src")),
        recursive = TRUE
      )

      # Create empty bin (for build log)
      dir.create(path = model_bin_path, showWarnings = FALSE, recursive = TRUE)

      # Build model
      if(self$args$os == "windows") {
        system2(
          command = self$args$shell,
          args = c(
            self$args$shell_flag,
            paste0(
              self$args$vs_cmd,
              " && ",
              "msbuild ",
              sln_path,
              " /t:Rebuild ",
              "/p:Configuration=",
              self$args$mode,
              " /p:OM_ROOT=",
              om_root,
              " /p:Platform=",
              self$args$bit,
              " & echo done > ",
              paste0(model_bin_path, "/build.done")
            )
          ),
          wait = FALSE,
          stdout = paste0(model_bin_path, "/build.log"),
          stderr = paste0(model_bin_path, "/build.log")
        )
      } else {
        system(
          command = paste0(
            "cd ",
            model_path,
            " && make RELEASE=",
            ifelse(test = self$args$mode == "Release", yes = 1, no = 0),
            " all publish > /dev/null 2>&1 &&",
            " echo \" Build succeeded.\" >> ", paste0(model_bin_path, "/build.log"), " &&",
            " touch ", paste0(model_bin_path, "/build.done ")
          ),
          wait = FALSE
        )
      }

      # Exit
      return(invisible(self))
    },

    cache_code = function() {
      # Create dir
      dir.create(
        path = paste0(self$args$dir, "/auditr-downloads"),
        showWarnings = FALSE
      )

      # Cache code
      if(file.exists(paste0(self$args$dir, "/auditr-downloads/code.rds"))) {
        code <- readRDS(paste0(self$args$dir, "/auditr-downloads/code.rds"))
        code[names(self$code)] <- self$code
        saveRDS(
          object = code,
          file = paste0(self$args$dir, "/auditr-downloads/code.rds")
        )
      } else {
        saveRDS(
          object = self$code,
          file = paste0(self$args$dir, "/auditr-downloads/code.rds")
        )
      }

      # Exit
      return(invisible(self))
    },

    .rebuild_openm = function(om_root) {
      # Clean up
      unlink(
        x = paste0(om_root, "/openm/", c("build.done", "build.log")),
        recursive = TRUE
      )

      # Find msbuild
      msbuild_path <- list.files(
        path = paste0(dirname(dirname(dirname(gsub('["\"]', '', self$args$vs_cmd)))), "/msbuild/current/bin"),
        pattern = "msbuild.exe$",
        full.names = TRUE,
        recursive = FALSE,
        ignore.case = TRUE
      )

      # Set paths for downstream use
      sln_path <- paste0(om_root, '/openm/openm.sln')

      # Rebuild openm
      withr::with_dir(
        new = dirname(msbuild_path),
        code = {
          system2(
            command = self$args$shell,
            args = c(
              self$args$shell_flag,
              paste0(
                "msbuild.exe ",
                sln_path,
                " /t:libopenm ",
                "/p:Configuration=",
                self$args$mode,
                " /p:OM_ROOT=",
                om_root,
                " /p:Platform=",
                self$args$bit,
                " & echo done > ",
                paste0(om_root, "/openm/build.done")
              )
            ),
            wait = FALSE,
            stdout = paste0(om_root, "/openm/build.log"),
            stderr = paste0(om_root, "/openm/build.log")
          )
        }
      )

      # Exit
      return(invisible(self))
    },

    git_token = c(),

    load_release = function(url, dest) {
      # Render console message
      cli::cli_alert("Downloading {.url {url}} and extracting to {.file {dest}}.")
      cli::cli_text("")

      # Create auditr-downloads folder
      dir.create(
        path = paste0(self$args$dir, "/auditr-downloads"),
        showWarnings = FALSE,
        recursive = TRUE
      )

      # Increase timeout limit to 10 minutes
      options(timeout = 600)

      # Download url
      if(file.exists(url)) {
        suppressWarnings(
          file.copy(
            from = url,
            to = paste0(self$args$dir, "/auditr-downloads/", basename(url)),
            recursive = FALSE
          )
        )
      } else {
        utils::download.file(
          url = url,
          destfile = paste0(self$args$dir, "/auditr-downloads/", basename(url)),
          method = "auto",
          quiet = TRUE
        )
      }

      # Remove destination path if it exists
      unlink(
        x = dest,
        recursive = TRUE,
        force = TRUE
      )

      # Create auditr-downloads folder
      dir.create(
        path = dest,
        showWarnings = FALSE,
        recursive = TRUE
      )

      # Extract url to dest
      archive::archive_extract(
        archive = paste0(self$args$dir, "/auditr-downloads/", basename(url)),
        dir = dest,
        strip_components = ifelse(
          test = self$args$os == "windows",
          yes = 0L,
          no = 1L
        )
      )

      # Exit
      return(invisible(self))
    },

    red = function(x) cli::make_ansi_style("#af3c43")(x),

    run_model = function(
      run_name = paste0("Model-run-", gsub(" |[:]", "-", Sys.time())),
      om_root,
      model_bin_path,
      table_grouping,
      parameters,
      sub_from,
      cases,
      threads,
      sub_samples,
      param_dir,
      db
    ) {
      # If db set
      if(! is.null(db)) {
        # Copy original database
        file.copy(
          from = paste0(model_bin_path, "/", self$args$model, "-copy.sqlite"),
          to = db,
          overwrite = TRUE
        )
      } else {
        # Get fresh copy of model database
        file.copy(
          from = paste0(model_bin_path, "/", self$args$model, "-copy.sqlite"),
          to = paste0(model_bin_path, "/", self$args$model, ".sqlite"),
          overwrite = TRUE
        )
      }

      # Check for other parameters
      if(! is.null(parameters)) {
        other_parameters <- sapply(
          X = names(parameters),
          FUN = function(x) {
            paste0(
              names(parameters[x]),
              "=",
              ifelse(
                test = parameters[x] %in% c("TRUE", "FALSE"),
                yes = tolower(parameters[x]),
                no = parameters[x]
              )
            )
          }
        ) |> unname()
      } else { other_parameters <- "" }

      # Check for sub_from parameters
      if(! is.null(sub_from)) {
        sub_from_parameters <- sapply(
          X = names(sub_from),
          FUN = function(x) {
            paste0(
              names(sub_from[x]),
              "=",
              ifelse(
                test = sub_from[x] %in% c("TRUE", "FALSE"),
                yes = tolower(sub_from[x]),
                no = sub_from[x]
              )
            )
          }
        ) |> unname()
        sub_from_parameters = c(
          "[SubFrom]",
          sub_from_parameters
        )
      } else { sub_from_parameters <- "" }

      # Build/store ini file
      writeLines(
        text = c(
          "[OpenM]",
          "SetName=Default",
          paste0("RunName=", run_name),
          paste0("SubValues=", sub_samples),
          paste0("Threads=", threads),
          ifelse(
            test = ! is.null(param_dir),
            yes = paste0("ParamDir=", param_dir),
            no = ""
          ),
          "",
          "[Parameter]",
          ifelse(
            test = ! is.null(cases),
            yes = paste0("SimulationCases=", sprintf("%.0f", cases)),
            no = ""
          ),
          other_parameters,
          "",
          sub_from_parameters,
          "",
          "[Tables]",
          paste0("Retain=", paste0(table_grouping, collapse = ","))
        ),
        con = paste0(model_bin_path, "/", self$args$model, ".ini")
      )

      # Clean up
      unlink(paste0(model_bin_path, "/run.done"))
      unlink(paste0(model_bin_path, "/run.log"))

      # Run model
      withr::with_dir(
        new = model_bin_path,
        code = {
          system2(
            command = self$args$shell,
            args = c(
              self$args$shell_flag,
              paste0(
                ifelse(
                  test = self$args$os == "windows",
                  yes = "set ",
                  no = "export "
                ),
                "OM_ROOT=",
                om_root,
                " && ",
                ifelse(
                  test = self$args$os != "windows",
                  yes = paste0("chmod +x ", model_bin_path, "/", self$args$model, " && ulimit -S -s 65536 && "),
                  no = ""
                ),
                self$args$model_exe,
                " -ini ",
                self$args$model,
                ".ini",
                ifelse(
                  test = ! is.null(db),
                  yes = " -db ",
                  no = ""
                ),
                ifelse(
                  test = ! is.null(db),
                  yes = db,
                  no = ""
                ),
                ifelse(
                  test = self$args$os != "windows",
                  yes = " > /dev/null 2>&1",
                  no = ""
                ),
                " && echo done > run.done"
              )
            ) |> unlist(),
            wait = FALSE,
            stdout = paste0(model_bin_path, "/run.log"),
            stderr = paste0(model_bin_path, "/run.log")
          )
        }
      )

      # Exit
      return(invisible(self))
    },

    strip_file_ext = function(path) {
      sub(
        pattern = "(\\.tar\\.gz|\\.tar\\.bz2|\\.tar\\.xz|\\.tgz|\\.zip|\\.7z|\\.git)$",
        replacement = "",
        x = basename(path),
        ignore.case = TRUE
      )
    },

    is_valid_code = function(x) {
      tryCatch({
        parse(text = x)
        TRUE
      }, error = function(e) {
        FALSE
      })
    }
  )
)
