# STCOpenMPP/OpenM++ Testing Automation

A Python tool for automating STCOpenMPP/OpenM++ model testing and comparison. This makes it easy to test your models across different STCOpenMPP/OpenM++ versions and see exactly what changed. Originally built by Parsa Abadi (first version available on [GitHub](https://github.com/parsaabadi/ompp-testing-automation/tree/main)), and maintained/developed further by Statistics Canada's Health Analysis and Modelling Division. 

## What This Does

If you work with STCOpenMPP models and need to test them across different versions, this tool handles the boring stuff:
- Grabs models from Git repositories
- Builds them with MSBuild
- Runs the models on different STCOpenMPP/OpenM++ versions
- Compares the results and shows you what's different
- Creates nice HTML reports you can actually read

The whole thing runs from command line and doesn't require you to click through GUIs or manually copy files around.

## What You Need

**System stuff:**
- Windows (because STCOpenMPP/OpenM++ and MSBuild)
- Python 3.8 or newer
- Visual Studio with MSBuild
- At least two STCOpenMPP/OpenM++ installations to compare

**Python packages:**
```bash
pip install -r requirements.txt
```

The main packages are pandas, click for the CLI, and a few others for Git operations.

## Getting Started

1. Change into this project directory
2. Run `pip install -r requirements.txt`
3. You're ready to go

## How to Use It

### The Easy Way - Full Workflow

Create a config file first:
```bash
python ompp_testing_cli.py create-config --output-file my-test.json
```

Edit the JSON file with your paths (STCOpenMPP/OpenM++ installations, Visual Studio, etc.), then:
```bash
python ompp_testing_cli.py run-test --config my-test.json
```

This does everything: clones the repo, builds models, runs them, compares results, and spits out an HTML report.

### The Manual Way - Step by Step

Clone a model repository:
```bash
python ompp_testing_cli.py clone --git-url "https://github.com/openmpp/main.git" --model-sln "riskpaths-ompp.sln"
```

Build it for your STCOpenMPP/OpenM++ versions:
```bash
python ompp_testing_cli.py build --model-sln "./path/to/model.sln" --om-root "C:/ompp/1.17.5" --om-root "C:/ompp/1.17.9" --vs-cmd-path "C:/Program Files/Microsoft Visual Studio/2022/Enterprise/Common7/Tools/VsDevCmd.bat"
```

See what output tables are available:
```bash
python ompp_testing_cli.py tables --model-name "RiskPaths" --om-root "C:/ompp/1.17.5"
```

### Using It in Your Own Scripts

```python
from ompp_testing import clone_repo, build_model, run_models, generate_html_report

# Clone and build
model_path = clone_repo("https://github.com/openmpp/main.git", model_sln="riskpaths-ompp.sln")
models = build_model(model_path, om_root=["C:/ompp/1.17.5", "C:/ompp/1.17.9"], vs_cmd_path="...")

# Run comparison
results = run_models(om_root=["C:/ompp/1.17.5", "C:/ompp/1.17.9"], model_name=models[0], cases=100000)

# Make a report
generate_html_report(results, title="My Test Results")
```

## Configuration

The config file is just JSON. Here's what it looks like:

```json
{
  "git_url": "https://github.com/openmpp/main.git",
  "git_commit": "9f4cf26ff8b7c4caf2b26621f02b4310a7380c2e",
  "model_sln": "riskpaths-ompp.sln",
  "om_root": [
    "C:/Users/yourname/Desktop/ompp/1.17.5",
    "C:/Users/yourname/Desktop/ompp/1.17.9"
  ],
  "vs_cmd_path": "C:/Program Files/Microsoft Visual Studio/2022/Enterprise/Common7/Tools/VsDevCmd.bat"
}
```

Change the paths to match your setup. You can leave `git_username` and `git_password` as null unless you need authentication.

## What You Get

**HTML Reports** - Clean, readable reports showing:
- Which tables were compared
- How many differences were found
- Statistics about the differences
- All the model run details

**Console Output** - Real-time progress with colors and progress bars so you know what's happening

**Data Files** - Pickle files with all the raw data if you want to do your own analysis

## When Things Go Wrong

**"Visual Studio not found"** - Check that VS 2022 or 2019 is installed and the path in your config is right

**"Model won't build"** - Verify your OM_ROOT paths and that you have all the dependencies

**"Git errors"** - Check your internet connection and Git credentials

Turn on debug mode if you need more details:
```bash
set OMPP_DEBUG=1
```

## File Structure

```
ompp_testing/
├── clone_repo.py         # Git operations
├── build_model.py        # MSBuild stuff  
├── get_output_tables.py  # Database queries
├── run_models.py         # Model execution
├── compare_model_runs.py # Compare results
└── report_generator.py   # HTML reports
```

Each module does one thing and does it well. You can import them individually if you want to build your own workflow.

## Examples

Check the `examples/` folder for working scripts. The RiskPaths example is a good place to start:

```bash
cd examples
python test_riskpaths.py
```

## Contributing

If you want to add features or fix bugs:
1. Add new modules in `ompp_testing/`
2. Update `__init__.py` to export new functions
3. Add CLI commands in `ompp_testing_cli.py` if needed
4. Test with real models before submitting

## License

MIT License - use it however you want.

## Thanks

Built for Statistics Canada's STCOpenMPP/OpenM++ testing needs. Thanks to the OpenM++ team for making a great foundation to build off of. 