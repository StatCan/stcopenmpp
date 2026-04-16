"""
HTML report generation functionality for STCOpenMPP testing.
"""

import os
from datetime import datetime
from pathlib import Path
import pandas as pd
from jinja2 import Template
import click


def generate_html_report(summary, output_tables, title="STCOpenMPP Testing Report", 
                        model_name=None, git_commit=None, om_versions=None, 
                        environment_note=None, output_dir=None):
    """
    Generate a nice HTML report from the comparison results.
    
    Takes all the comparison data and makes a clean, readable HTML report
    that you can send to others or keep for your records.
    """
    
    # Prepare data for the template
    data = _prepare_report_data(
        summary, output_tables, title, model_name, 
        git_commit, om_versions, environment_note
    )
    
    # Render the HTML
    html_content = _render_html_template(data)
    
    # Determine output file path
    if output_dir:
        output_dir = Path(output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)
        output_file = output_dir / f"ompp_testing_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.html"
    else:
        output_file = f"ompp_testing_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.html"
    
    # Save the report
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(html_content)
        
    return str(output_file)


def _prepare_report_data(summary, output_tables, title, model_name, 
                        git_commit, om_versions, environment_note):
    """Get all the data ready for the HTML template."""
    
    if summary is None:
        return {
            'title': title,
            'error': 'No summary data available'
        }
    
    comparison_results = summary.get('comparison_results', [])
    summary_table = summary.get('summary_table', pd.DataFrame())
    original_results = summary.get('original_results', [])
    
    tables_with_diffs = [r for r in comparison_results if r.get('has_differences', False)]
    tables_without_diffs = [r for r in comparison_results if not r.get('has_differences', False)]
    
    return {
        'title': title,
        'model_name': model_name,
        'git_commit': git_commit,
        'om_versions': om_versions,
        'environment_note': environment_note,
        'generation_time': datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        'total_tables': len(comparison_results),
        'tables_with_differences': len(tables_with_diffs),
        'tables_without_differences': len(tables_without_diffs),
        'summary_table': summary_table,
        'comparison_results': comparison_results,
        'original_results': original_results,
        'output_tables': output_tables
    }


def _render_html_template(data):
    """Turn the data into HTML using a Jinja2 template."""
    
    template = Template("""
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ data.title }}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #2c3e50;
            border-bottom: 3px solid #3498db;
            padding-bottom: 10px;
        }
        h2 {
            color: #34495e;
            margin-top: 30px;
        }
        .summary-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .stat-card {
            background: #ecf0f1;
            padding: 20px;
            border-radius: 6px;
            text-align: center;
        }
        .stat-number {
            font-size: 2em;
            font-weight: bold;
            color: #2980b9;
        }
        .stat-label {
            color: #7f8c8d;
            margin-top: 5px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #3498db;
            color: white;
        }
        tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        .diff-yes {
            color: #e74c3c;
            font-weight: bold;
        }
        .diff-no {
            color: #27ae60;
            font-weight: bold;
        }
        .error {
            color: #e74c3c;
            font-style: italic;
        }
        .metadata {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            margin: 20px 0;
        }
        .metadata h3 {
            margin-top: 0;
            color: #2c3e50;
        }
        .metadata p {
            margin: 5px 0;
        }
        .version-info {
            display: inline-block;
            background: #3498db;
            color: white;
            padding: 5px 10px;
            border-radius: 4px;
            margin: 2px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{ data.title }}</h1>
        
        <div class="metadata">
            <h3>Test Information</h3>
            {% if data.model_name %}
            <p><strong>Model:</strong> {{ data.model_name }}</p>
            {% endif %}
            {% if data.om_versions %}
            <p><strong>STCOpenMPP Versions:</strong> 
                {% for version in data.om_versions.split(' vs ') %}
                <span class="version-info">{{ version }}</span>
                {% endfor %}
            </p>
            {% endif %}
            {% if data.git_commit %}
            <p><strong>Git Commit:</strong> {{ data.git_commit[:8] }}</p>
            {% endif %}
            {% if data.environment_note %}
            <p><strong>Environment:</strong> {{ data.environment_note }}</p>
            {% endif %}
            <p><strong>Generated:</strong> {{ data.generation_time }}</p>
        </div>
        
        <div class="summary-stats">
            <div class="stat-card">
                <div class="stat-number">{{ data.total_tables }}</div>
                <div class="stat-label">Total Tables</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{ data.tables_with_differences }}</div>
                <div class="stat-label">Tables with Differences</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{ data.tables_without_differences }}</div>
                <div class="stat-label">Tables without Differences</div>
            </div>
        </div>
        
        {% if data.summary_table is defined and not data.summary_table.empty %}
        <h2>Summary of Differences</h2>
        <table>
            <thead>
                <tr>
                    <th>Table Name</th>
                    <th>Version Comparison</th>
                    <th>Has Differences</th>
                    <th>Total Values</th>
                    <th>Differences</th>
                    <th>Difference %</th>
                </tr>
            </thead>
            <tbody>
                {% for _, row in data.summary_table.iterrows() %}
                <tr>
                    <td>{{ row.table_name }}</td>
                    <td>{{ row.version_comparison }}</td>
                    <td class="{% if row.has_differences %}diff-yes{% else %}diff-no{% endif %}">
                        {% if row.has_differences %}Yes{% else %}No{% endif %}
                    </td>
                    <td>{{ row.total_values }}</td>
                    <td>{{ row.total_differences }}</td>
                    <td>{{ row.difference_percent }}</td>
                </tr>
                {% endfor %}
            </tbody>
        </table>
        {% endif %}
        
        {% if data.comparison_results %}
        <h2>Detailed Comparison Results</h2>
        {% for comparison in data.comparison_results %}
        <h3>{{ comparison.table_name }}</h3>
        {% if comparison.has_differences %}
        <p><strong>Status:</strong> <span class="diff-yes">Differences Found</span></p>
        {% for version, diff_stats in comparison.differences.items() %}
        <h4>Comparison: {{ comparison.base_version }} vs {{ version }}</h4>
        {% if 'error' in diff_stats %}
        <p class="error">Error: {{ diff_stats.error }}</p>
        {% else %}
        <ul>
            <li><strong>Total Values:</strong> {{ "{:,}".format(diff_stats.total_values) }}</li>
            <li><strong>Total Differences:</strong> {{ "{:,}".format(diff_stats.total_differences) }}</li>
            <li><strong>Difference Percentage:</strong> {{ "{:.2f}%".format(diff_stats.difference_percent) }}</li>
        </ul>
        {% if diff_stats.column_differences %}
        <h5>Column Differences:</h5>
        <table>
            <thead>
                <tr>
                    <th>Column</th>
                    <th>Difference Count</th>
                    <th>Difference %</th>
                    <th>Min Diff</th>
                    <th>Max Diff</th>
                    <th>Median Diff</th>
                </tr>
            </thead>
            <tbody>
                {% for col, col_stats in diff_stats.column_differences.items() %}
                <tr>
                    <td>{{ col }}</td>
                    <td>{{ "{:,}".format(col_stats.diff_count) }}</td>
                    <td>{{ "{:.2f}%".format(col_stats.diff_percent) }}</td>
                    <td>{{ "{:.6f}".format(col_stats.min_diff) }}</td>
                    <td>{{ "{:.6f}".format(col_stats.max_diff) }}</td>
                    <td>{{ "{:.6f}".format(col_stats.median_diff) }}</td>
                </tr>
                {% endfor %}
            </tbody>
        </table>
        {% endif %}
        {% endif %}
        {% endfor %}
        {% else %}
        <p><strong>Status:</strong> <span class="diff-no">No Differences Found</span></p>
        {% endif %}
        <hr>
        {% endfor %}
        {% endif %}
    </div>
</body>
</html>
    """)
    
    return template.render(data=data)


def generate_summary_stats(summary):
    """
    Generate summary statistics from the results.
    
    Args:
        summary (dict): Summary results from run_models
        
    Returns:
        dict: Summary statistics
    """
    stats = {
        'total_tables': 0,
        'tables_with_differences': 0,
        'total_comparisons': 0,
        'successful_comparisons': 0
    }
    
    if 'output_table_summary' in summary and not summary['output_table_summary'].empty:
        df = summary['output_table_summary']
        stats['total_tables'] = len(df)
        
        # Count tables with differences (non-zero diff_count)
        if 'diff_count' in df.columns:
            # Handle formatted strings (with commas) and convert to int
            diff_counts = []
            for val in df['diff_count']:
                try:
                    # Remove commas and convert to int
                    clean_val = str(val).replace(',', '').replace('-', '0')
                    diff_counts.append(int(clean_val))
                except (ValueError, TypeError):
                    diff_counts.append(0)
            
            stats['tables_with_differences'] = sum(1 for count in diff_counts if count > 0)
        
        stats['successful_comparisons'] = len(df[df['unique_value_digests'] != 'Error'])
    
    if 'model_run_summary' in summary and not summary['model_run_summary'].empty:
        # Count unique model runs
        if 'model_run_number' in summary['model_run_summary'].columns:
            stats['total_comparisons'] = len(
                summary['model_run_summary']['model_run_number'].unique()
            )
    
    return stats 