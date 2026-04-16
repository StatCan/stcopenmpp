#!/usr/bin/env python3
"""
STCOpenMPP Testing Automation

Command-line tools for testing STCOpenMPP models across different versions.
"""

import os
import sys
import json
from pathlib import Path
import click
import pandas as pd
import time

# Add the ompp_testing package to the path
sys.path.insert(0, str(Path(__file__).parent))

from ompp_testing import (
    clone_repo, build_model, run_models, get_output_tables, generate_html_report
)


@click.group()
@click.version_option(version='1.0.0')
def cli():
    """
    STCOpenMPP Testing Automation
    
    Tools for testing STCOpenMPP models across different versions.
    """
    pass


@cli.command()
@click.option('--config', '-c', type=click.Path(exists=True), 
              help='Path to configuration JSON file')
@click.option('--git-url', help='Git repository URL')
@click.option('--git-username', help='Git username for authentication')
@click.option('--git-password', help='Git password/token for authentication')
@click.option('--git-commit', help='Specific commit to checkout')
@click.option('--model-sln', help='Model solution file name')
@click.option('--om-root', multiple=True, help='STCOpenMPP root directories')
@click.option('--vs-cmd-path', help='Path to Visual Studio command prompt')
@click.option('--cases', default=1000000, type=int, help='Number of simulation cases')
@click.option('--threads', default=8, type=int, help='Number of threads')
@click.option('--sub-samples', default=8, type=int, help='Number of sub-samples')
@click.option('--tables-per-run', default=25, type=int, help='Tables per run batch')
@click.option('--max-run-time', default=86400, type=int, help='Maximum time to wait for each model run in seconds (default: 86400 = 24 hours)')
@click.option('--output-dir', help='Output directory for reports')
def run_test(config, git_url, git_username, git_password, git_commit, model_sln, 
             om_root, vs_cmd_path, cases, threads, sub_samples, tables_per_run, max_run_time, output_dir):
    """
    Run the complete testing workflow.
    
    This does everything: clones the repo, builds models, runs them on different 
    STCOpenMPP versions, compares results, and makes an HTML report.
    """
    
    ## Record runtime for entire program to finish. 
    program_start = time.perf_counter()
    
    # Load configuration from file if provided
    settings = {}
    if config:
        with open(config, 'r') as f:
            settings = json.load(f)
    
    # Override with command line arguments
    if git_url:
        settings['git_url'] = git_url
    if git_username:
        settings['git_username'] = git_username
    if git_password:
        settings['git_password'] = git_password
    if git_commit:
        settings['git_commit'] = git_commit
    if model_sln:
        settings['model_sln'] = model_sln
    if om_root:
        settings['om_root'] = list(om_root)
    if vs_cmd_path:
        settings['vs_cmd_path'] = vs_cmd_path
    
    # Validate required settings
    required_settings = ['git_url', 'model_sln', 'om_root', 'vs_cmd_path']
    missing_settings = [s for s in required_settings if s not in settings]
    if missing_settings:
        click.echo(f"Missing required settings: {', '.join(missing_settings)}")
        click.echo("Please provide them via command line options or configuration file.")
        sys.exit(1)
    
    try:
        click.echo("\nStarting STCOpenMPP testing workflow...\n")
        
        # Clone repository
        click.echo("Cloning repository...")
        model_sln_path = clone_repo(
            git_url=settings['git_url'],
            git_username=settings.get('git_username'),
            git_password=settings.get('git_password'),
            git_commit=settings.get('git_commit'),
            model_sln=settings['model_sln']
        )
        
        # Build models
        click.echo("\nBuilding models...")
        model_names = build_model(
            model_sln=model_sln_path,
            om_root=settings['om_root'],
            vs_cmd_path=settings['vs_cmd_path'],
            mode="release",
            bit=64
        )
        
        if not model_names:
            raise RuntimeError("No models were built successfully\n")
        
        model_name = model_names[0]  # Use first model name
        
        # Get output tables
        click.echo("\nGetting output table list...")
        output_tables = get_output_tables(
            model_name=model_name,
            om_root=settings['om_root'][0]
        )
        
        # Run models and compare
        results = run_models(
            om_root=settings['om_root'],
            model_name=model_name,
            cases=cases,
            threads=threads,
            sub_samples=sub_samples,
            tables=None,
            tables_per_run=tables_per_run,
            max_run_time=max_run_time
        )
        
        # Generate report
        click.echo("\nGenerating HTML report...")
        
        report_path = generate_html_report(
            summary=results,
            output_tables=output_tables,
            title=f"STCOpenMPP Testing Report - {model_name}",
            model_name=model_name,
            git_commit=settings.get('git_commit'),
            om_versions=' vs '.join([Path(p).name for p in settings['om_root']]),
            environment_note=f"Python testing tool - {len(settings['om_root'])} versions compared",
            output_dir=output_dir
        )
        ## Pickle file (.pkl) with actual data from the above comparison. Commented out to save storage; uncomment if you want to do analysis on the compared output tables. 
        """# Try to save the results for later analysis
        try:
            import pickle
            from datetime import datetime
            
            results_file = f"results_{model_name}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.pkl"
            with open(results_file, 'wb') as f:
                pickle.dump({
                    'results': results,
                    'output_tables': output_tables,
                    'settings': settings
                }, f)
        except Exception:
            pass  # Don't fail if we can't save results"""

        click.echo(f"   Testing workflow completed successfully!")
        click.echo(f"   Report available at: {report_path}")

        ## End, process, and print recorded runtime for entire program.
        program_end = time.perf_counter()
        elapsed_program_time = program_end - program_start
        formatted_runtime = f"{int(elapsed_program_time // 3600):02d}:{int((elapsed_program_time % 3600) // 60):02d}:{round(elapsed_program_time % 60)}"
        click.echo(f"   Testing finished in {formatted_runtime}.")

        # Show quick summary
        if results and results.get('summary_table') is not None:
            summary_df = results['summary_table']
            if not summary_df.empty:
                total_tables = len(summary_df)
                tables_with_diffs = len(summary_df[summary_df['has_differences'] == True])
                click.echo(f"\nSummary: {total_tables} tables analyzed, {tables_with_diffs} with differences.\n")
        
    except Exception as e:
        click.echo(f"   Testing workflow failed: {str(e)}")
        sys.exit(1)


@cli.command()
@click.option('--git-url', required=True, help='Git repository URL')
@click.option('--git-username', help='Git username for authentication')
@click.option('--git-password', help='Git password/token for authentication')
@click.option('--git-commit', help='Specific commit to checkout')
@click.option('--model-sln', required=True, help='Model solution file name')
def clone(git_url, git_username, git_password, git_commit, model_sln):
    """Clone a repository and find the model solution file."""
    try:
        model_sln_path = clone_repo(
            git_url=git_url,
            git_username=git_username,
            git_password=git_password,
            git_commit=git_commit,
            model_sln=model_sln
        )
        click.echo(f"Model solution file found at: {model_sln_path}")
    except Exception as e:
        click.echo(f"Clone failed: {str(e)}")
        sys.exit(1)


@cli.command()
@click.option('--model-sln', required=True, help='Path to model solution file')
@click.option('--om-root', multiple=True, required=True, help='STCOpenMPP root directories')
@click.option('--vs-cmd-path', required=True, help='Path to Visual Studio command prompt')
@click.option('--mode', default='release', help='Build mode (release/debug)')
@click.option('--bit', default=64, type=int, help='Target architecture (64/32)')
def build(model_sln, om_root, vs_cmd_path, mode, bit):
    """Build models for specified STCOpenMPP versions."""
    try:
        model_names = build_model(
            model_sln=model_sln,
            om_root=list(om_root),
            vs_cmd_path=vs_cmd_path,
            mode=mode,
            bit=bit
        )
        click.echo(f"Built models: {', '.join(model_names)}")
    except Exception as e:
        click.echo(f"Build failed: {str(e)}")
        sys.exit(1)


@cli.command()
@click.option('--model-name', required=True, help='Name of the model')
@click.option('--om-root', required=True, help='STCOpenMPP root directory')
def tables(model_name, om_root):
    """List available output tables for a model."""
    try:
        output_tables = get_output_tables(model_name=model_name, om_root=om_root)
        
        click.echo(f"Found {len(output_tables)} output tables:")
        for idx, row in output_tables.iterrows():
            desc = row.get('description', 'No description')
            if pd.isna(desc) or desc == 'No description available':
                click.echo(f"  {idx + 1:2d}. {row['name']}")
            else:
                click.echo(f"  {idx + 1:2d}. {row['name']:<30} {desc}")
        
    except Exception as e:
        click.echo(f"Failed to get tables: {str(e)}")
        sys.exit(1)


@cli.command()
@click.option('--output-file', default='config.json', help='Output configuration file name')
def create_config(output_file):
    """Create a sample configuration file."""
    
    sample_config = {
        "git_url": "https://github.com/openmpp/main.git",
        "git_username": None,
        "git_password": None,
        "git_commit": "latest",
        "model_sln": "model-ompp.sln",
        "om_root": [
            "c:/path/to/ompp/version1",
            "c:/path/to/ompp/version2"
        ],
        "vs_cmd_path": "c:/program files/microsoft visual studio/2022/enterprise/common7/tools/vsdevcmd.bat"
    }
    
    config_file = Path(output_file)
    with open(config_file, 'w') as f:
        json.dump(sample_config, f, indent=2)
    
    click.echo(f"Created sample configuration file: {config_file}")
    click.echo("Edit this file with your specific paths and settings.")


if __name__ == '__main__':
    cli() 