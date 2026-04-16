# Create package structure
usethis::create_package(path = getwd())

# Create DESCRIPTION file
usethis::use_description(
  fields = list(
    Package = "auditr",
    Version = "0.31",
    Title = "R6 methods for auditing two releases of stcopenmpp",
    Description = "This package provides R6 methods for auditing two releases of stcopenmpp.",
    `Authors@R` = c(
      utils::person(
        given = "Joel",
        family = "Barnes",
        email = "joel.barnes@statcan.gc.ca",
        role = c("aut", "cre")
      )
    ),
    License = "MIT"
  )
)

# Add dependencies to DESCRIPTION file
usethis::use_package("archive", "Imports")
usethis::use_package("cli", "Imports")
usethis::use_package("DBI", "Imports")
usethis::use_package("dplyr", "Imports")
usethis::use_package("jsonlite", "Imports")
usethis::use_package("kableExtra", "Suggests")
usethis::use_package("knitr", "Imports")
usethis::use_package("quarto", "Imports")
usethis::use_package("readr", "Imports")
usethis::use_package("R6", "Imports")
usethis::use_package("rlang", "Imports")
usethis::use_package("RSQLite", "Imports")
usethis::use_package("sodium", "Imports")
usethis::use_package("stringr", "Imports")
usethis::use_package("styler", "Imports")
usethis::use_package("tidyr", "Imports")
usethis::use_package("utils", "Imports")
usethis::use_package("withr", "Imports")

# Create NAMESPACE file
usethis::use_namespace()

# Create license
usethis::use_mit_license()

# Create tests folder
usethis::use_testthat()

# Create tests
usethis::use_test("")

# Run tests
devtools::test()

# Update NAMESPACE and create documentation
devtools::check_man()

# Check package build
devtools::check()
