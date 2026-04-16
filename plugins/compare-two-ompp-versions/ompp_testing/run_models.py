"""
Model execution and comparison functionality for STCOpenMPP testing.
"""

import os
from git import rmtree
import shutil
import time
import json
from datetime import datetime
from pathlib import Path
import pandas as pd
import click
import subprocess

from .compare_model_runs import compare_model_runs
from .get_output_tables import get_table_data



def run_models(om_root, model_name, cases=1000000, threads=8, sub_samples=8, 
               tables=None, tables_per_run=25, max_run_time=86400):
    """
    Run models on different STCOpenMPP versions and compare the results.
    
    This is the main function that does the heavy lifting. It runs your model.exe 
    with the parameters you specify, and then compares the output tables between 
    different versions.
    
    The function is designed to handle long-running models that may take hours
    or even days to complete.
    
    Args:
        max_run_time: Maximum time to wait for each model run in seconds (default: 86400 = 24 hours)
                     Set to None for unlimited waiting time (not recommended)
    """
    click.echo(f"   Starting model runs for {model_name}")
    click.echo(f"       Cases: {cases:,}, Threads: {threads}, Sub-samples: {sub_samples}")
    click.echo(f"       Maximum run time per version: {max_run_time//3600:.1f} hours")
    
    if not tables:
        click.echo(f"       No tables specified, will get all output tables")
        from .get_output_tables import get_output_tables
        output_tables = get_output_tables(model_name, om_root[0])
        tables = output_tables['name'].tolist()
    
    click.echo(f"       Will compare {len(tables)} tables across {len(om_root)} STCOpenMPP versions")
    
    all_results = []
    
    try:
        ## Record runtime for all model runs to finish. 
        models_run_start = time.perf_counter()
       
        ## Run model over each OM_ROOT version to later compare outputs.
        for i, root in enumerate(om_root):
            click.echo(f"\nRunning model for version {Path(root).name}...")
            
            version_results = _run_single_version(  
                root, model_name, cases, threads, sub_samples, 
                tables, tables_per_run, i, max_run_time
            )
             
            all_results.append(version_results)
        
        ## End, process, and print recorded runtime for all model runs. 
        models_run_end = time.perf_counter()
        models_run_elapsed = models_run_end - models_run_start
        formatted_models_runtime = f"{int(models_run_elapsed // 3600):02d}:{int((models_run_elapsed % 3600) // 60):02d}:{round(models_run_elapsed % 60)}"
        click.echo(f"   All model runs finished in {formatted_models_runtime}.")

        ## Function _run_single_version changes current directory to respective om_root models/bin folder; change back so report is generated in this project's root.
        current_path = Path(__file__).resolve().parent.parent
        os.chdir(current_path)
        from .compare_model_runs import compare_model_runs
        comparison = compare_model_runs(all_results)
        
        return comparison
        
    except Exception as e:
        click.echo(f"Model run failed: {str(e)}")
        raise

    finally:
        cloned_folder = Path(__file__).resolve().parent.parent / model_name
        if os.path.exists(cloned_folder) and os.path.isdir(cloned_folder):
            rmtree(cloned_folder)


def _debug_model_files(om_root, model_name):
    """Debug function to check what model files exist and their properties."""
    models_bin = Path(om_root) / 'models' / 'bin'
    model_specific_bin = Path(om_root) / 'models' / model_name / 'ompp' / 'bin'
    
    click.echo(f"       Checking models directory: {models_bin}")
    if models_bin.exists():
        files = list(models_bin.iterdir())
        click.echo(f"           Found {len(files)} files in models/bin:")
        for f in files:
            if f.suffix in ['.exe', '.sqlite']:
                click.echo(f"               {f.name} ({f.stat().st_size} bytes)")
    else:
        click.echo(f"       Models directory does not exist: {models_bin}")
    
    click.echo(f"       Checking model-specific directory: {model_specific_bin}")
    if model_specific_bin.exists():
        files = list(model_specific_bin.iterdir())
        click.echo(f"       Found {len(files)} files in model-specific bin:")
        for f in files:
            if f.suffix in ['.exe', '.sqlite']:
                click.echo(f"           {f.name} ({f.stat().st_size} bytes)")
    else:
        click.echo(f"       Model-specific directory does not exist: {model_specific_bin}")


def _fix_model_detection(om_root, model_name):
    """Try to fix model detection by ensuring database files are in the right place."""
    models_bin = Path(om_root) / 'models' / 'bin'
    model_specific_bin = Path(om_root) / 'models' / model_name / 'ompp' / 'bin'
    
    sqlite_file = f"{model_name}.sqlite"
    exe_file = f"{model_name}.exe"
    
    target_sqlite = models_bin / sqlite_file
    target_exe = models_bin / exe_file
    
    source_locations = [
        model_specific_bin / sqlite_file,
        model_specific_bin / exe_file,
        Path(om_root) / 'bin' / sqlite_file,
        Path(om_root) / 'bin' / exe_file
    ]
    
    click.echo(f"       Ensuring model files are in service directory...")
    
    files_copied = False
    
    for source_file in source_locations:
        if source_file.exists():
            if source_file.suffix == '.sqlite':
                if not target_sqlite.exists() or target_sqlite.stat().st_size == 0:
                    try:
                        shutil.copy2(source_file, target_sqlite)
                        click.echo(f"      Copied {source_file} -> {target_sqlite}")
                        files_copied = True
                    except Exception as e:
                        click.echo(f"      Failed to copy {source_file}: {e}")
            
            elif source_file.suffix == '.exe':
                if not target_exe.exists():
                    try:
                        shutil.copy2(source_file, target_exe)
                        click.echo(f"      Copied {source_file} -> {target_exe}")
                        files_copied = True
                    except Exception as e:
                        click.echo(f"      Failed to copy {source_file}: {e}")
    
    if files_copied:
        click.echo(f"    Files copied, waiting for service to refresh...")
        time.sleep(3)
        return model_name
    
    exe_files = list(models_bin.glob("*.exe"))
    if exe_files:
        target_exe = f"{model_name}.exe"
        if any(f.name.lower() == target_exe.lower() for f in exe_files):
            click.echo(f"       Found target executable: {target_exe}, using model name: {model_name}")
            return model_name
        else:
            detected_name = exe_files[0].stem
            click.echo(f"       Target model not found, using first executable: {exe_files[0].name}")
            click.echo(f"       Available executables: {[f.name for f in exe_files]}")
            click.echo(f"       Using model name: {detected_name}")
            return detected_name
    
    return None


def _run_single_version(om_root, model_name, cases, threads, sub_samples, 
                       tables, tables_per_run, version_index, max_run_time):
    
    try:
        model_names = []  # Initialize to avoid reference errors
        
        click.echo(f"   Debugging model detection...")
        _debug_model_files(om_root, model_name)
        
        click.echo(f"   Starting model run...")

        actual_model_name = _fix_model_detection(om_root, model_name)
        run_request = {
            "ModelName": actual_model_name,
            "SetName": "Default",
            "RunName": f"TestRun_{int(time.time())}",
            "SimulationCases": str(cases),
            "Threads": str(threads),
            "SubValues": str(sub_samples),
            "Tables": ",".join(tables)
        }

        os.chdir(om_root+"/models/bin/")
        run_request_cmd = f".\\{run_request["ModelName"]}.exe -OpenM.SetName {run_request["SetName"]} -OpenM.RunName {run_request["RunName"]} -OpenM.Threads {run_request["Threads"]} -OpenM.SubValues {run_request["SubValues"]} -Parameter.SimulationCases {run_request["SimulationCases"]} -Tables.Retain {run_request["Tables"]}"
        
        ## Record runtime for single model run to finish. 
        model_run_start = time.perf_counter()

        ## Run model.exe command.
        subprocess.Popen(
            run_request_cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True
        )
        ## subprocess.run(run_request_cmd, capture_output=True, text=True, check=True)  ## Alternative run command. Waits until command done so no progress log can be printed using this option.

        ## Set up progress log loop for model.exe run. To-do: add run time count and display at the end. One timer for model run, one timer for entire program. 
        click.echo(f"       Model run progress:")
        prev_log_found = ""
        while True:
            # Check if log file exists
            log_path = Path(os.getcwd()) / f"{run_request['ModelName']}.log"
            if log_path.exists():
                output = subprocess.check_output(
                    [
                        "powershell",
                        "-Command",
                        f"Get-Content '{log_path}' -Tail 10"
                    ],
                    text=True
                ).split("\n")[-2]
                
                # If no new update to latest log message, then skip printing progress.
                if output != prev_log_found:
                    click.echo(f"           Latest log line: {output}")  # Print progress.
                prev_log_found = output  # Store latest log message to compare in next loop.

                # Check if "Done."
                if "Done." in output:
                    break

        ## End, process, and print recorded runtime for all model runs. 
        model_run_end = time.perf_counter()
        model_run_elapsed = model_run_end - model_run_start
        formatted_model_runtime = f"{int(model_run_elapsed // 3600):02d}:{int((model_run_elapsed % 3600) // 60):02d}:{round(model_run_elapsed % 60)}"
        click.echo(f"       {actual_model_name} run with {om_root} finished in {formatted_model_runtime}.")

        table_data = _get_all_table_data(om_root, actual_model_name, tables, tables_per_run)

        tables_retrieved = len([v for v in table_data.values() if v is not None])
        click.echo(f"   Successfully retrieved {tables_retrieved}/{len(tables)} tables")
        
        return {
            'version': Path(om_root).name,
            'run_request': run_request,
            'actual_model_name': actual_model_name,
            'table_data': table_data
        }
        
    except Exception as e:
        click.echo(f"Model run failed: {str(e)}")
        raise


def _get_all_table_data(om_root, model_name, tables, tables_per_run, run_id=None):
    """Get data from all output tables for a model run."""
    
    table_data = {}
    
    for i in range(0, len(tables), tables_per_run):
        batch = tables[i:i + tables_per_run]
        
        click.echo(f"    Getting batch {i//tables_per_run + 1}/{(len(tables) + tables_per_run - 1)//tables_per_run}")
        
        for table_name in batch:
            try:
                data = get_table_data(model_name, om_root, table_name, run_id)
                table_data[table_name] = data
            except Exception as e:
                click.echo(f"    Failed to get {table_name}: {str(e)}")
                table_data[table_name] = None
    
    return table_data