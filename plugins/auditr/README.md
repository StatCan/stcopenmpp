

## Overview

`auditr` is a cross-platform (Windows and Linux) R package that provides
R6 methods for auditing two releases of
<a href="https://github.com/statcan/stcopenmpp" target="_blank">stcopenmpp</a>.
The methods in this package allow users to download and extract two
releases of `stcopenmpp`, clone and checkout a model from any point in
its history, compile the model asynchronously in both releases, run the
model asynchronously in both releases and generate an audit report in
Quarto (`.html`).

`auditr` interacts with model `.exe` files directly and relies on SQL
queries to interact with model `.sqlite` files. This design eliminates
the need for the OpenM++ web service and, consequently, reduces time
significantly when doing model runs. `auditr` also cleans up model
databases immediately after model runs and audits to avoid any
unexpected incursions on storage capacity in the case of models with
large and/or numerous output tables.

## Installation

``` r
# Set working directory
setwd("path/to/auditr")

# Load auditr
devtools::load_all()
```

## Usage

### Auditing two releases with one model

``` r
# Initialize release_audit R6 class
audit <- release_audit$new(
  releases = c(
    "https://github.com/StatCan/stcopenmpp/releases/download/v0.2/stcopenmpp-windows-v0.2-gh-release.zip",
    "https://github.com/StatCan/stcopenmpp/releases/download/v0.3/stcopenmpp-windows-v0.3-gh-release.zip"
  ),
  labels = c(
    "stcopenmpp v0.2",
    "stcopenmpp v0.3"
  ),
  sln = "RiskPaths-ompp.sln",
  mode = "release",
  bit = 64,
  git_url = "https://github.com/openmpp/main.git",
  git_commit = "v1.17.15",
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
  cases = 1e6,
  threads = 4,
  sub_samples = 4,
  tables_per_run = 5
)

# Generate Quarto report
audit$load_report()
```

#### Audit report

Download the report in `.html`:
[inst/html/demo-auditr-report.html](inst/html/demo-auditr-report.html?inline=false)

#### Audit meta

``` r
# View audit meta
audit$results$audit
```

    #> # A tibble: 7 × 10
    #>   table                        table_size expression_count dimension_count
    #>   <chr>                        <chr>      <chr>            <chr>          
    #> 1 T01_LifeExpectancy           3          3                0              
    #> 2 T02_TotalPopulationByYear    2 x 101    2                1              
    #> 3 T03_FertilityByAge           2 x 27     2                1              
    #> 4 T04_FertilityRatesByAgeGroup 1 x 12 x 6 1                2              
    #> 5 T05_CohortFertility          3          3                0              
    #> 6 T06_BirthsByUnion            1 x 7      1                1              
    #> 7 T07_FirstUnionFormation      1 x 12     1                1              
    #>   value_count diff_count diff_percent min_diff max_diff median_diff
    #>   <chr>       <chr>      <chr>        <chr>    <chr>    <chr>      
    #> 1 3           0          0.00         -        -        -          
    #> 2 202         0          0.00         -        -        -          
    #> 3 54          0          0.00         -        -        -          
    #> 4 72          0          0.00         -        -        -          
    #> 5 3           0          0.00         -        -        -          
    #> 6 7           0          0.00         -        -        -          
    #> 7 12          0          0.00         -        -        -

##### Model run meta

``` r
# View model run meta
audit$results$model_runs
```

    #> # A tibble: 32 × 3
    #>    Variable      `stcopenmpp v0.1`              `stcopenmpp v0.2`             
    #>    <chr>         <chr>                          <chr>                         
    #>  1 run_number    1                              2                             
    #>  2 run_id        102                            102                           
    #>  3 model_id      101                            101                           
    #>  4 run_stamp     2026_07_22_09_45_55_487        2026_07_22_09_45_55_495       
    #>  5 run_digest    37e215b37c39fc...20dcc47bf850e f66b147571de07...ab612df3e0b7e
    #>  6 create_dt     2026-07-22 09:45:55.547        2026-07-22 09:45:55.547       
    #>  7 update_dt     2026-07-22 09:46:17.507        2026-07-22 09:46:16.964       
    #>  8 cases         1e+06                          1e+06                         
    #>  9 sub_count     4                              4                             
    #> 10 sub_started   4                              4                             
    #> 11 sub_restart   4                              4                             
    #> 12 sub_completed 4                              4                             
    #> 13 status        s                              s                             
    #> 14 value_digest  e56784b5fdf5de...3ff4d5c37bd5a e56784b5fdf5de...3ff4d5c37bd5a
    #> 15 release_path  C:/Users/barnj...-windows-v0.1 C:/Users/barnj...-windows-v0.2
    #> 16 output_tables T01_LifeExpect...hortFertility T01_LifeExpect...hortFertility
    #> 17 run_number    3                              4                             
    #> 18 run_id        102                            102                           
    #> 19 model_id      101                            101                           
    #> 20 run_stamp     2026_07_22_09_46_20_416        2026_07_22_09_46_20_432       
    #> 21 run_digest    7e6a4def9cedaf...d875df47eed80 f93796529f9356...af6c83c2b63f1
    #> 22 create_dt     2026-07-22 09:46:20.439        2026-07-22 09:46:20.458       
    #> 23 update_dt     2026-07-22 09:46:38.713        2026-07-22 09:46:38.622       
    #> 24 cases         1e+06                          1e+06                         
    #> 25 sub_count     4                              4                             
    #> 26 sub_started   4                              4                             
    #> 27 sub_restart   4                              4                             
    #> 28 sub_completed 4                              4                             
    #> 29 status        s                              s                             
    #> 30 value_digest  b248a5a663b2f5...9d8eb62a27347 b248a5a663b2f5...9d8eb62a27347
    #> 31 release_path  C:/Users/barnj...-windows-v0.1 C:/Users/barnj...-windows-v0.2
    #> 32 output_tables T06_BirthsByUn...nionFormation T06_BirthsByUn...nionFormation

##### Output tables

``` r
# View list of non-hidden output tables
audit$results$output_tables
```

    #> # A tibble: 7 × 3
    #>   id    name                         label                              
    #>   <chr> <chr>                        <chr>                              
    #> 1 101   T01_LifeExpectancy           Life Expectancy                    
    #> 2 102   T02_TotalPopulationByYear    Life table                         
    #> 3 103   T03_FertilityByAge           Age-specific fertility             
    #> 4 104   T04_FertilityRatesByAgeGroup Fertility rates by age group       
    #> 5 105   T05_CohortFertility          Cohort fertility                   
    #> 6 106   T06_BirthsByUnion            Pregnancies by union status & order
    #> 7 107   T07_FirstUnionFormation      First union formation
