"""
Output table retrieval functionality for STCOpenMPP testing.
"""

import sqlite3
import pandas as pd
from pathlib import Path
import click


def get_output_tables(model_name, om_root):
    """
    Get a list of output tables available for a model.
    
    Connects to the STCOpenMPP database and pulls out all the output tables
    that are available for your model. This tells you what data you can
    compare between different STCOpenMPP versions.
    """
    
    db_paths = [
        Path(om_root) / 'models' / f'{model_name}.sqlite',
        Path(om_root) / 'models' / f'{model_name}.db',
        Path(om_root) / 'models' / 'bin' / f'{model_name}.sqlite',
        Path(om_root) / 'models' / 'bin' / f'{model_name}.db'
    ]
    
    db_path = None
    for path in db_paths:
        if path.exists():
            db_path = path
            break
    
    if not db_path:
        raise FileNotFoundError(f"Can't find the model database. Tried: {[str(p) for p in db_paths]}")
    
    try:
        conn = sqlite3.connect(str(db_path))
        
        # First get the basic table info from table_dic
        table_dic_query = """
            SELECT
                l.table_hid,
                l.table_name AS name,
                l.table_digest AS digest
            FROM table_dic AS l
            LEFT JOIN model_table_dic AS r
                ON l.table_hid = r.table_hid
            WHERE
                r.is_hidden = 0
            ORDER BY
                l.table_name;
        """
        
        tables_df = pd.read_sql_query(table_dic_query, conn)
        
        # Try to get descriptions from table_dic_txt (like the R version does)
        try:
            desc_query = """
            SELECT 
                table_hid,
                descr as description
            FROM table_dic_txt
            WHERE lang_id = 0
            """
            
            desc_df = pd.read_sql_query(desc_query, conn)
            
            # Join tables with descriptions
            tables = pd.merge(tables_df, desc_df, on='table_hid', how='left')
            
        except Exception as desc_e:
            # If table_dic_txt doesn't exist or query fails, just use basic info
            click.echo(f"  Note: Could not get table descriptions: {str(desc_e)}")
            tables = tables_df
            tables['description'] = 'No description available'
        
        # Clean up - remove table_hid since we don't need it in the final result
        if 'table_hid' in tables.columns:
            tables = tables.drop('table_hid', axis=1)
        
        conn.close()
        
        click.echo(f"   Found {len(tables)} output tables for {model_name} in {om_root}")
        
        return tables
        
    except Exception as e:
        click.echo(f"  Failed to get tables: {str(e)}")
        raise


def get_table_data(model_name, om_root, table_name, run_id=None):
    """
    Get the actual data from a specific output table.
    
    If you don't specify a run_id, it'll get the latest run.
    Returns the table data as a pandas DataFrame.
    """
    db_paths = [
        Path(om_root) / 'models' / f'{model_name}.sqlite',
        Path(om_root) / 'models' / f'{model_name}.db',
        Path(om_root) / 'models' / 'bin' / f'{model_name}.sqlite',
        Path(om_root) / 'models' / 'bin' / f'{model_name}.db'
    ]
    
    db_path = None
    for path in db_paths:
        if path.exists():
            db_path = path
            break
    
    if not db_path:
        raise FileNotFoundError(f"Can't find the model database. Tried: {[str(p) for p in db_paths]}")
    
    try:
        conn = sqlite3.connect(str(db_path))
        
        if run_id is None:
            run_query = "SELECT MAX(run_id) as latest_run FROM run_lst"
            latest_run = pd.read_sql_query(run_query, conn)
            run_id = latest_run.iloc[0]['latest_run']
        else:
            # API RunStamp is timestamp format, but database uses integer run_ids
            # Always use the latest run_id since we just initiated a new run
            try:
                run_query = "SELECT MAX(run_id) as latest_run FROM run_lst"
                latest_run = pd.read_sql_query(run_query, conn)
                if not latest_run.empty and latest_run.iloc[0]['latest_run'] is not None:
                    actual_run_id = latest_run.iloc[0]['latest_run']
                    click.echo(f"    Using latest database run_id {actual_run_id} instead of API RunStamp {run_id}")
                    run_id = actual_run_id
            except Exception as e:
                click.echo(f"    Could not get latest run_id, using provided: {e}")
        
        # Debug: Check what's actually in the database. Debug outputs commented out for log readability. Uncomment to re-enable debug logs for table retrieval.
        """try:
            # Check available runs
            run_check = pd.read_sql_query("SELECT * FROM run_lst ORDER BY run_id DESC LIMIT 5", conn)
            click.echo(f"    Debug: Found {len(run_check)} recent runs in database")
            if len(run_check) > 0:
                click.echo(f"    Debug: Latest run_ids: {run_check['run_id'].tolist()}")
                click.echo(f"    Debug: Looking for run_id: {run_id}")
            
            # Check if table exists and has data (will be done after actual_table_name is determined)
            pass
            
        except Exception as debug_e:
            click.echo(f"    Debug: Could not check database contents: {debug_e}")"""
        
        # Check if table exists, and if not, look for table with digest suffix
        actual_table_name = table_name
        try:
            table_exists_query = f"SELECT name FROM sqlite_master WHERE type='table' AND name='{table_name}'"
            table_exists = pd.read_sql_query(table_exists_query, conn)
            
            if table_exists.empty:
                # Look for table with digest suffix (common in STCOpenMPP databases)
                suffix_query = f"SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '{table_name}_%'"
                suffix_tables = pd.read_sql_query(suffix_query, conn)
                
                if not suffix_tables.empty:
                    actual_table_name = suffix_tables.iloc[0]['name']
                    click.echo(f"       Using table with digest: {actual_table_name} for {table_name}")
                else:
                    conn.close()
                    raise Exception(f"Table {table_name} does not exist in database (with or without digest suffix)")
            
        except Exception as table_check_e:
            conn.close()
            raise Exception(f"Could not verify table existence: {table_check_e}")
        
        # Check table row count for debugging. Debug outputs commented out for log readability. Uncomment to re-enable debug logs for table retrieval.
        """try:
            table_check = pd.read_sql_query(f"SELECT COUNT(*) as count FROM {actual_table_name}", conn)
            total_rows = table_check.iloc[0]['count']
            click.echo(f"    Debug: Table {actual_table_name} has {total_rows} total rows")
        except Exception as count_e:
            click.echo(f"    Debug: Could not count rows in {actual_table_name}: {count_e}")"""
        
        # Try different run identification approaches based on STCOpenMPP version
        queries_to_try = [
            f"SELECT * FROM {actual_table_name} WHERE run_id = ?",
            f"SELECT * FROM {actual_table_name} WHERE run_digest = ?", 
            f"SELECT * FROM {actual_table_name} WHERE run_name = ?",
            f"SELECT * FROM {actual_table_name} ORDER BY ROWID DESC LIMIT 1000"
        ]
        
        data = None
        successful_query = None
        
        for i, query in enumerate(queries_to_try):
            try:
                if "LIMIT" in query:
                    data = pd.read_sql_query(query, conn)
                    if len(data) > 0:
                        successful_query = f"fallback query (latest {len(data)} rows)"
                else:
                    data = pd.read_sql_query(query, conn, params=[str(run_id)])
                    if len(data) > 0:
                        query_types = ["run_id", "run_digest", "run_name"]
                        successful_query = f"{query_types[i]} match"
                
                if data is not None and len(data) > 0:
                    click.echo(f"           Debug: Successfully retrieved {len(data)} rows using {successful_query}")
                    break
                    
            except Exception as query_e:
                continue
        
        conn.close()
        
        if data is None or len(data) == 0:
            raise Exception(f"No data found for table {table_name} with run_id {run_id}")
        
        return data
        
    except Exception as e:
        click.echo(f"  Failed to get table data: {str(e)}")
        return None


def get_model_runs(model_name, om_root):
    """
    Get a list of all model runs in the database.
    
    Shows you what runs are available and when they were created.
    """
    db_paths = [
        Path(om_root) / 'models' / f'{model_name}.sqlite',
        Path(om_root) / 'models' / f'{model_name}.db',
        Path(om_root) / 'models' / 'bin' / f'{model_name}.sqlite',
        Path(om_root) / 'models' / 'bin' / f'{model_name}.db'
    ]
    
    db_path = None
    for path in db_paths:
        if path.exists():
            db_path = path
            break
    
    if not db_path:
        raise FileNotFoundError(f"Can't find the model database. Tried: {[str(p) for p in db_paths]}")
    
    try:
        conn = sqlite3.connect(str(db_path))
        
        query = """
        SELECT 
            run_id,
            run_name,
            run_digest,
            create_dt,
            update_dt,
            run_status
        FROM run_lst
        ORDER BY create_dt DESC
        """
        
        runs = pd.read_sql_query(query, conn)
        conn.close()
        
        return runs
        
    except Exception as e:
        click.echo(f"  Failed to get model runs: {str(e)}")
        return None 