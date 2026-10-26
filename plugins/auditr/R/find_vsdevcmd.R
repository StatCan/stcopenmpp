#' @noRd
find_vsdevcmd <- function(require_cpp = TRUE) {
  pf86 <- Sys.getenv("ProgramFiles(x86)", "C:/Program Files (x86)")
  pf <- Sys.getenv("ProgramFiles", "C:/Program Files")

  # 1. Ask vswhere (most reliable)
  vswhere <- file.path(
    pf86,
    "Microsoft Visual Studio",
    "Installer",
    "vswhere.exe"
  )

  if (file.exists(vswhere)) {
    args <- c("-latest", "-products", "*", "-prerelease")
    if (require_cpp) {
      args <- c(
        args,
        "-requires",
        "Microsoft.VisualStudio.Component.VC.Tools.x86.x64"
      )
    }
    args <- c(args, "-property", "installationPath")
    inst <- suppressWarnings(
      system2(vswhere, args, stdout = TRUE, stderr = FALSE)
    )
    inst <- inst[nzchar(inst)]
    if (length(inst)) {
      bat <- file.path(inst[1], "Common7", "Tools", "VsDevCmd.bat")
      if (file.exists(bat)) return(normalizePath(bat, winslash = "\\"))
    }
  }

  # 2. Fallback: search the usual install locations
  candidates <- Sys.glob(
    c(
      file.path(
        pf,
        "Microsoft Visual Studio",
        "*",
        "*",
        "Common7",
        "Tools",
        "VsDevCmd.bat"
      ),
      file.path(
        pf86,
        "Microsoft Visual Studio",
        "*",
        "*",
        "Common7",
        "Tools",
        "VsDevCmd.bat"
      )
    )
  )

  if (length(candidates)) {
    # Prefer the newest version folder (e.g. 18 over 2022)
    ver <- basename(dirname(dirname(dirname(dirname(candidates)))))
    ord <- order(
      suppressWarnings(as.numeric(ver)) %in% NA,
      -suppressWarnings(as.numeric(ver))
    )
    return(normalizePath(candidates[ord][1], winslash = "\\"))
  }

  stop("VsDevCmd.bat not found. Is Visual Studio (with C++ workload) installed?")
}
