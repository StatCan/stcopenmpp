#!/usr/bin/env python3
"""
Example script for testing RiskPaths model with different STCOpenMPP versions.
"""

import sys
from pathlib import Path

# Add the parent directory to the path so we can import ompp_testing
sys.path.insert(0, str(Path(__file__).parent.parent))

from ompp_testing import (
    clone_repo, build_model, run_models, 
    get_output_tables, generate_html_report,
    generate_summary_stats
)


def test_riskpaths():
    """Test RiskPaths model across different STCOpenMPP versions."""
    
    print("Starting RiskPaths model testing")
    
    # Configuration
    config = {
        'git_url': 'https://github.com/openmpp/main.git',
        'git_commit': '9f4cf26ff8b7c4caf2b26621f02b4310a7380c2e',
        'model_sln': 'riskpaths-ompp.sln',
        'om_root': [
            'c:/users/username/desktop/ompp/1.17.5',
            'c:/users/username/desktop/ompp/1.17.9'
        ],
        'vs_cmd_path': 'c:/program files/microsoft visual studio/2022/enterprise/common7/tools/vsdevcmd.bat'
    }
    
    try:
        # Clone repository
        print("\nCloning repository...")
        model_sln_path = clone_repo(
            git_url=config['git_url'],
            git_commit=config['git_commit'],
            model_sln=config['model_sln']
        )
        
        # Build model
        print("\nBuilding model...")
        model_names = build_model(
            model_sln=model_sln_path,
            om_root=config['om_root'],
            vs_cmd_path=config['vs_cmd_path'],
            mode="release",
            bit=64
        )
        
        if not model_names:
            raise RuntimeError("No models were built successfully")
        
        model_name = model_names[0]
        print(f"Model built successfully: {model_name}")
        
        # Get output tables
        print("\nGetting output tables...")
        output_tables = get_output_tables(
            model_name=model_name,
            om_root=config['om_root'][0]
        )
        
        print(f"Found {len(output_tables)} output tables")
        
        # Run models and compare
        print("\nRunning models and comparing results...")
        results = run_models(
            om_root=config['om_root'],
            model_name=model_name,
            cases=100000,  # Smaller test run
            threads=4,
            sub_samples=4,
            tables=None,
            tables_per_run=10,
            max_run_time=7200  # 2 hours for example runs
        )
        
        # Generate report
        print("\nGenerating HTML report...")
        report_path = generate_html_report(
            summary=results,
            output_tables=output_tables,
            title="RiskPaths Testing Report",
            model_name=model_name,
            git_commit=config['git_commit'],
            om_versions=' vs '.join([Path(p).name for p in config['om_root']]),
            environment_note="Example Python testing script"
        )
        
        print(f"\nTesting completed successfully!")
        print(f"Report available at: {report_path}")
        
        # Show summary
        if results and results.get('summary_table') is not None:
            summary_df = results['summary_table']
            if not summary_df.empty:
                total_tables = len(summary_df)
                tables_with_diffs = len(summary_df[summary_df['has_differences'] == True])
                print(f"Summary: {total_tables} tables analyzed, {tables_with_diffs} with differences")
        
        return results
        
    except Exception as e:
        print(f"Testing failed: {str(e)}")
        raise


if __name__ == '__main__':
    test_riskpaths() 