"""
STCOpenMPP Testing Automation

A Python toolkit for testing STCOpenMPP models across different versions.
"""

from .clone_repo import clone_repo
from .build_model import build_model
from .get_output_tables import get_output_tables, get_table_data, get_model_runs
from .run_models import run_models
from .compare_model_runs import compare_model_runs
from .report_generator import generate_html_report, generate_summary_stats

__version__ = "1.0.0"
__author__ = "Statistics Canada"

__all__ = [
    'clone_repo',
    'build_model', 
    'get_output_tables',
    'get_table_data',
    'get_model_runs',
    'run_models',
    'compare_model_runs',
    'generate_html_report',
    'generate_summary_stats'
] 