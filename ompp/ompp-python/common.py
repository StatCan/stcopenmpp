import os
import sys
import sqlite3
import csv
import re
import pandas as pd
from openpyxl import Workbook

def run_sqlite_script(db, sql_script):

    try:
        conn = sqlite3.connect(db)
        cursor = conn.cursor()
        with open(sql_script, 'r') as script_file:
            script_content = script_file.read()
            cursor.executescript(script_content)
        conn.commit()
        return True
    except Exception as e:
        logmsg(error(), "run_sqlite_script", f"Failed to run the script on '{db}': {e}")
        return False
    finally:
        if 'conn' in locals():
            conn.close()

def write_to_dat_file(table_name, columns, data, output_file):
 
    try:
        with open(output_file, 'w') as outfile:
            outfile.write(f"# Data for table: {table_name}\n")
            outfile.write(f"# Columns: {', '.join(columns)}\n")
            for row in data:
                formatted_row = " ".join(map(str, row))
                outfile.write(f"{formatted_row}\n")
        logmsg(info(), "write_to_dat_file", f"Data written to '{output_file}'")
    except Exception as e:
        logmsg(error(), "write_to_dat_file", f"Failed to write data to '{output_file}': {e}")

def export_to_excel(input_db, output_excel):
    """Export all tables from an SQLite database to an Excel file."""
    try:
        # Check if the database file exists
        if not os.path.exists(input_db):
            logmsg(error(), "Database Error", f"Input database '{input_db}' not found.")
            return False

        conn = sqlite3.connect(input_db)
        logmsg(info(), "Database", f"Connected to database '{input_db}'.")

        with pd.ExcelWriter(output_excel, engine='openpyxl') as writer:
            cursor = conn.cursor()
            cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
            tables = [row[0] for row in cursor.fetchall()]

            for table_name in tables:
                logmsg(info(), "Database", f"Exporting table '{table_name}' to Excel sheet.")
                table_data = pd.read_sql_query(f"SELECT * FROM {table_name}", conn)
                table_data.to_excel(writer, sheet_name=table_name, index=False)

        logmsg(info(), "Excel Export", f"Successfully exported database '{input_db}' to Excel file '{output_excel}'.")
        return True

    except Exception as e:
        logmsg(error(), "Excel Export Error", f"Failed to export '{input_db}' to '{output_excel}': {str(e)}")
        return False
    finally:
        if 'conn' in locals():
            conn.close()
            logmsg(info(), "Database", "Database connection closed.")
def warning():
    return "  warning => "

def change():
    return "DIFFERS ===> "

def error():
    return "ERROR =====> "

def info():
    return "             "

def diagnostic():
    return "diagnostic   "

def logmsg(prefix, *args):
 
    message = args[-1]
    prefixes = [prefix]
    for part in args[:-1]:
        if part == '':
            continue
        part = re.sub(r'( *)$', r':\1 ', part)
        prefixes.append(part)
    full_prefix = ''.join(prefixes)
    for line in message.split('\n'):
        print(f"{full_prefix}{line}")

def apply_transform(value, round_prec):
    transformed_value = value * 1.12345678987654321
    if round_prec > 0:
        transformed_value = float(f"{transformed_value:.15g}")
        transformed_value = float(f"{transformed_value:.{round_prec}g}")
    return transformed_value

def ompp_tables_to_csv(db, dir, round_prec=0, zero_fuzz=1e-15, do_original=0, do_transformed=0):
    rounding_on = False
    if round_prec > 0:
        rounding_on = True

    dir = dir.replace('\\', '/')

    dir_original = f"{dir}/original"
    dir_transformed = f"{dir}/transformed"

    outdirs = [dir]
    if do_original:
        outdirs.append(dir_original)
    if do_transformed:
        outdirs.append(dir_transformed)

    for fldr in outdirs:
        if not os.path.isdir(fldr):
            try:
                os.makedirs(fldr)
            except Exception as e:
                logmsg(error(), f"unable to create directory {fldr}")
                return 1

    try:
        conn = sqlite3.connect(db)
        cursor = conn.cursor()
    except sqlite3.Error as e:
        logmsg(error(), f"Cannot connect to database {db}: {e}")
        return 1

    try:
        cursor.execute("Select table_name, table_rank From table_dic Order By table_name;")
        tables_data = cursor.fetchall()
    except sqlite3.Error as e:
        logmsg(error(), f"Failed to retrieve table list: {e}")
        conn.close()
        return 1

    tables = []
    ranks = {}
    for col1, col2 in tables_data:
        tables.append(col1)
        ranks[col1] = col2

    for table in tables:
        rank = ranks[table]
        order_clause = "Order By " + ','.join([f"Dim{dim}" for dim in range(rank+1)])
        select_query = f"Select * From {table} {order_clause};"

        try:
            cursor.execute(select_query)
            rows = cursor.fetchall()
            columns = [description[0] for description in cursor.description]
        except sqlite3.Error as e:
            logmsg(error(), f"Failed to retrieve data from table {table}: {e}")
            conn.close()
            return 1

        if len(rows) == 0:
            continue

        out_csv = f"{dir}/{table}.csv"
        out_csv_original = f"{dir_original}/{table}.csv" if do_original else None
        out_csv_transformed = f"{dir_transformed}/{table}.csv" if do_transformed else None

        try:
            with open(out_csv, 'w', newline='') as outfile:
                writer = csv.writer(outfile)
                writer.writerow(columns)

                if do_original:
                    out_original = open(out_csv_original, 'w', newline='')
                    writer_original = csv.writer(out_original)
                    writer_original.writerow(columns)
                if do_transformed:
                    out_transformed = open(out_csv_transformed, 'w', newline='')
                    writer_transformed = csv.writer(out_transformed)
                    writer_transformed.writerow(columns)

                for row in rows:
                    row = list(row)
                    if len(row) == 0:
                        continue
                    if row[-1] is not None and row[-1] != '':
                        value = row[-1]
                        try:
                            original_value = float(value)
                        except ValueError:
                            original_value = value
                            value = value
                            transformed_value = value
                        else:
                            if abs(original_value) <= zero_fuzz:
                                value = 0.0
                            else:
                                value = original_value
                            if rounding_on:
                                value = float(f"{value:.15g}")
                                value = float(f"{value:.{round_prec}g}")
                            if do_transformed:
                                transformed_value = apply_transform(value, round_prec)
                            else:
                                transformed_value = value

                            value_str = f"{value:.15g}"
                            value_str = re.sub(r'e([-+])0(\d\d)', r'e\1\2', value_str)
                            if do_original:
                                original_value_str = f"{original_value:.15g}"
                                original_value_str = re.sub(r'e([-+])0(\d\d)', r'e\1\2', original_value_str)
                            if do_transformed:
                                transformed_value_str = f"{transformed_value:.15g}"
                                transformed_value_str = re.sub(r'e([-+])0(\d\d)', r'e\1\2', transformed_value_str)
                        row_out = row[:-1] + [value_str]
                        writer.writerow(row_out)
                        if do_original:
                            row_original = row[:-1] + [original_value_str]
                            writer_original.writerow(row_original)
                        if do_transformed:
                            row_transformed = row[:-1] + [transformed_value_str]
                            writer_transformed.writerow(row_transformed)
                    else:
                        writer.writerow(row)
                        if do_original:
                            writer_original.writerow(row)
                        if do_transformed:
                            writer_transformed.writerow(row)

                if do_original:
                    out_original.close()
                if do_transformed:
                    out_transformed.close()

        except Exception as e:
            logmsg(error(), f"Error processing table {table}: {e}")
            conn.close()
            return 1

    conn.close()
    return 0

def read_parameters_from_csv(csv_file):
    
    parameters = {}
    with open(csv_file, 'r', newline='') as infile:
        reader = csv.DictReader(infile)
        fieldnames = reader.fieldnames

        required_fields = {'name', 'type', 'value'}
        if not required_fields.issubset(set(fieldnames)):
            raise ValueError("CSV file must contain 'name', 'type', and 'value' columns")

        is_array = 'index' in fieldnames

        for row in reader:
            name = row['name']
            param_type = row['type']
            value = row['value']
            index = row.get('index')

            if name not in parameters:
                parameters[name] = {'type': param_type, 'values': [], 'indices': [] if is_array else None}

            if is_array:
                if index is None:
                    raise ValueError(f"Missing 'index' for parameter '{name}'")
                parameters[name]['indices'].append(int(index))
                parameters[name]['values'].append(value)
            else:
                parameters[name]['values'] = value 

    return parameters

def write_parameters_to_dat(parameters, dat_file):
    """
    Writes parameters to a DAT file.

    Parameters is a dictionary as returned by read_parameters_from_csv.
    """
    with open(dat_file, 'w') as outfile:
        for name, param in parameters.items():
            param_type = param['type']
            values = param['values']
            indices = param['indices']

            if indices is None:
                outfile.write(f"{param_type} {name} = {values};\n")
            else:
               
                max_index = max(indices)
                array_size = max_index + 1
                sorted_values = [None] * array_size
                for idx, val in zip(indices, values):
                    sorted_values[idx] = val
                outfile.write(f"{param_type} {name}[{array_size}] = {{\n")
                for i, val in enumerate(sorted_values):
                    outfile.write(f"    {val}")
                    if i < array_size -1:
                        outfile.write(",\n")
                    else:
                        outfile.write("\n")
                outfile.write("};\n")
    return
