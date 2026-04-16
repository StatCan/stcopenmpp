

## Getting started

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

### Loading auditr

``` r
# Set working directory
setwd("path/to/auditr")

# Load auditr
devtools::load_all()
```

### Using auditr

``` r
# Initialize auditr R6 class
audit <- auditr$new(
  releases = c(
    "https://github.com/openmpp/main/releases/download/v1.17.9/openmpp_win_20250601.zip",
    "https://github.com/openmpp/main/releases/download/v1.17.10/openmpp_win_20250923.zip"
  ),
  labels = c(
    "OpenM++ v1.17.9",
    "OpenM++ v1.17.10"
  ),
  sln = "RiskPaths-ompp.sln",
  mode = "release",
  bit = 64,
  git_url = "https://github.com/openmpp/main.git",
  git_commit = "v1.17.9",
  vs_cmd = "c:/program files/microsoft visual studio/18/professional/common7/tools/vsdevcmd.bat",
  platform_toolset = "v145"
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

#### Model run meta

``` r
# View model run meta
audit$results$model_runs
```

    #> # A tibble: 32 × 3
    #>    Variable      `OpenM++ v1.17.9`              `OpenM++ v1.17.10`            
    #>    <chr>         <chr>                          <chr>                         
    #>  1 run_number    1                              2                             
    #>  2 run_id        102                            102                           
    #>  3 run_stamp     2026_01_26_08_37_44_686        2026_01_26_08_37_45_371       
    #>  4 run_digest    828e0b4ddf2a83...44d1c67441ebf 7032002bf830d7...53a762871c66d
    #>  5 create_dt     2026-01-26 08:37:44.709        2026-01-26 08:37:45.39        
    #>  6 update_dt     2026-01-26 08:38:05.812        2026-01-26 08:38:06.222       
    #>  7 cases         1e+06                          1e+06                         
    #>  8 sub_count     4                              4                             
    #>  9 sub_started   4                              4                             
    #> 10 sub_completed 4                              4                             
    #> 11 status        s                              s                             
    #> 12 value_digest  e56784b5fdf5de...3ff4d5c37bd5a e56784b5fdf5de...3ff4d5c37bd5a
    #> 13 release_path  C:/Users/barnj..._win_20250601 C:/Users/barnj..._win_20250923
    #> 14 output_tables T01_LifeExpect...hortFertility T01_LifeExpect...hortFertility
    #> 15 descr         Default                        Default                       
    #> 16 lang_code     EN                             EN                            
    #> 17 run_number    3                              4                             
    #> 18 run_id        102                            102                           
    #> 19 run_stamp     2026_01_26_08_38_11_611        2026_01_26_08_38_11_616       
    #> 20 run_digest    a2aaba2f09ec01...b155dffc84d2b 8610d6750f623f...8292374f1da94
    #> 21 create_dt     2026-01-26 08:38:11.633        2026-01-26 08:38:11.638       
    #> 22 update_dt     2026-01-26 08:38:29.916        2026-01-26 08:38:32.576       
    #> 23 cases         1e+06                          1e+06                         
    #> 24 sub_count     4                              4                             
    #> 25 sub_started   4                              4                             
    #> 26 sub_completed 4                              4                             
    #> 27 status        s                              s                             
    #> 28 value_digest  b248a5a663b2f5...9d8eb62a27347 b248a5a663b2f5...9d8eb62a27347
    #> 29 release_path  C:/Users/barnj..._win_20250601 C:/Users/barnj..._win_20250923
    #> 30 output_tables T06_BirthsByUn...nionFormation T06_BirthsByUn...nionFormation
    #> 31 descr         Default                        Default                       
    #> 32 lang_code     EN                             EN

#### Output tables

``` r
# View list of non-hidden output tables
audit$results$output_tables
```

    #> # A tibble: 7 × 3
    #>   id    name                         label                              
    #>   <chr> <chr>                        <chr>                              
    #> 1 0     T01_LifeExpectancy           Life Expectancy                    
    #> 2 1     T02_TotalPopulationByYear    Life table                         
    #> 3 2     T03_FertilityByAge           Age-specific fertility             
    #> 4 3     T04_FertilityRatesByAgeGroup Fertility rates by age group       
    #> 5 4     T05_CohortFertility          Cohort fertility                   
    #> 6 5     T06_BirthsByUnion            Pregnancies by union status & order
    #> 7 6     T07_FirstUnionFormation      First union formation
