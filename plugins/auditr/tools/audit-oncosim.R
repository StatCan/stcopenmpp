# Load auditr
devtools::load_all()

# Initialize release_audit R6 class
audit <- release_audit$new(
  releases = c(
    config::get("release1_url"),
    config::get("release2_url")
  ),
  labels = c(
    "stcopenmpp v0.2",
    "stcopenmpp v0.3"
  ),
  sln = "OncoSimX-ompp.sln",
  mode = "release",
  bit = 64,
  git_url = config::get("oncosim_git_url"),
  git_user = config::get("oncosim_git_user"),
  git_token = config::get("oncosim_git_token"),
  git_commit = config::get("oncosim_git_commit"),
  keep_vcxproj = TRUE,
  vs_cmd = find_vsdevcmd(),
  platform_toolset = find_msvc_toolset()
)

# Download and extract releases
audit$load_releases()

# Download model
audit$load_model()

# Compile model in both releases asynchronously
audit$build_models()

# Get list of model's non-hidden output tables
audit$load_output_tables()

# Initiate model runs and audit output tables in both releases asynchronously
audit$run_models(
  cases = 1e4,
  threads = 4,
  sub_samples = 4,
  tables_per_run = 50
)

# Generate Quarto report
audit$load_report()
