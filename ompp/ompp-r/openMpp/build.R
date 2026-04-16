# Create package structure
usethis::create_package(path = getwd())

# Create DESCRIPTION file
usethis::use_description(
  fields = list(
    Package = "openMpp",
    Version = "0.8.8",
    Title = "Read and write OpenM++ database from R.",
    Description = "Set input parameters and get output results of openM++ models using R.",
    `Authors@R` = c(
      utils::person(
        given = "Anatoly",
        family = "Cherkassky",
        email = "openmpp99@gmail.com",
        role = c("aut", "cre")
      ),
      utils::person(
        given = "Maikol",
        family = "Diasparra",
        email = "maikol.diasparra@statcan.gc.ca",
        role = c("ctb")
      ),
      utils::person(
        given = "Claude",
        family = "Nadeau",
        email = "claude.nadeau@statcan.gc.ca",
        role = c("ctb")
      )
    ),
    License = "MIT"
  )
)

# Add dependencies to DESCRIPTION file
usethis::use_package("DBI", "Imports")
usethis::use_package("RODBCDBI", "Imports")
usethis::use_package("RSQLite", "Imports")

# Create NAMESPACE file
usethis::use_namespace()

# Create license
usethis::use_mit_license()

# Update NAMESPACE and create documentation
devtools::check_man()

# Create tests folder
usethis::use_testthat()

# Create tests
usethis::use_test("multiplication-works")

# Run tests
devtools::test()

# Check package build
devtools::check()

# Build package
devtools::build()
