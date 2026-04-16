"""
Model run comparison functionality for STCOpenMPP testing.
"""

import pandas as pd
import numpy as np
from pathlib import Path
import click


def compare_model_runs(results):
    """
    Compare model run results between different STCOpenMPP versions.
    
    Takes the results from running the same model on different STCOpenMPP versions
    and compares the output tables to see what's different. This is where you
    find out if upgrading STCOpenMPP changed your results.
    """
    if len(results) < 2:
        click.echo("Need at least 2 versions to compare")
        return None
    
    click.echo("\nComparing output tables between versions...")
    
    all_tables = set()
    for result in results:
        if result and result.get('table_data'):
            all_tables.update(result['table_data'].keys())
    
    click.echo(f"   Found {len(all_tables)} tables to compare")
    
    comparison_results = []
    
    for table_name in sorted(all_tables):
        click.echo(f"   Comparing {table_name}...")
        
        table_comparison = _compare_single_table(results, table_name)
        if table_comparison:
            comparison_results.append(table_comparison)
    
    summary = _create_summary(comparison_results, results)
    
    click.echo(f"   Comparison complete. Found differences in {len([r for r in comparison_results if r['has_differences']])} tables")
    
    return summary


def _compare_single_table(results, table_name):
    """Compare one specific table across all versions."""
    
    table_data = {}
    
    for result in results:
        if result and result.get('table_data') and table_name in result['table_data']:
            table_data[result['version']] = result['table_data'][table_name]
    
    if len(table_data) < 2:
        return None
    
    versions = list(table_data.keys())
    base_version = versions[0]
    other_versions = versions[1:]
    
    comparison = {
        'table_name': table_name,
        'base_version': base_version,
        'other_versions': other_versions,
        'has_differences': False,
        'differences': {}
    }
    
    base_data = table_data[base_version]
    
    for other_version in other_versions:
        other_data = table_data[other_version]
        
        if base_data is None or other_data is None:
            comparison['differences'][other_version] = {
                'error': 'Missing data for comparison'
            }
            continue
        
        diff_stats = _calculate_differences(base_data, other_data)
        comparison['differences'][other_version] = diff_stats
        
        if diff_stats['has_differences']:
            comparison['has_differences'] = True
    
    return comparison


def _calculate_differences(base_data, other_data):
    """Calculate the differences between two datasets."""
    
    if base_data.shape != other_data.shape:
        return {
            'has_differences': True,
            'error': f'Shape mismatch: {base_data.shape} vs {other_data.shape}'
        }
    
    numeric_cols = base_data.select_dtypes(include=[np.number]).columns.tolist()
    
    if not numeric_cols:
        return {
            'has_differences': False,
            'error': 'No numeric columns to compare'
        }
    
    differences = {}
    total_diffs = 0
    total_values = 0
    
    for col in numeric_cols:
        if col in other_data.columns:
            base_vals = base_data[col].fillna(0)
            other_vals = other_data[col].fillna(0)
            
            diff = other_vals - base_vals
            abs_diff = np.abs(diff)
            
            col_diffs = (abs_diff > 1e-10).sum()
            total_diffs += col_diffs
            total_values += len(diff)
            
            if col_diffs > 0:
                differences[col] = {
                    'diff_count': int(col_diffs),
                    'diff_percent': float(col_diffs / len(diff) * 100),
                    'min_diff': float(diff.min()),
                    'max_diff': float(diff.max()),
                    'median_diff': float(diff.median()),
                    'mean_diff': float(diff.mean())
                }
    
    return {
        'has_differences': total_diffs > 0,
        'total_values': total_values,
        'total_differences': total_diffs,
        'difference_percent': float(total_diffs / total_values * 100) if total_values > 0 else 0,
        'column_differences': differences
    }


def _create_summary(comparison_results, original_results):
    """Create a summary of all the comparisons."""
    
    summary_data = []
    
    for comparison in comparison_results:
        table_name = comparison['table_name']
        has_diffs = comparison['has_differences']
        
        for other_version, diff_stats in comparison['differences'].items():
            if 'error' in diff_stats:
                summary_data.append({
                    'table_name': table_name,
                    'version_comparison': f"{comparison['base_version']} vs {other_version}",
                    'has_differences': has_diffs,
                    'error': diff_stats['error'],
                    'total_values': '-',
                    'total_differences': '-',
                    'difference_percent': '-'
                })
            else:
                summary_data.append({
                    'table_name': table_name,
                    'version_comparison': f"{comparison['base_version']} vs {other_version}",
                    'has_differences': has_diffs,
                    'error': None,
                    'total_values': f"{diff_stats['total_values']:,}",
                    'total_differences': f"{diff_stats['total_differences']:,}",
                    'difference_percent': f"{diff_stats['difference_percent']:.2f}%"
                })
    
    summary_df = pd.DataFrame(summary_data)
    
    return {
        'comparison_results': comparison_results,
        'summary_table': summary_df,
        'original_results': original_results
    } 