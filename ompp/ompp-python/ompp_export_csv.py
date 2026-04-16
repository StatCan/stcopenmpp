import argparse
import os
import sys

import common  

def main():
    script_name = 'ompp_export_csv'
    script_version = '1.0'

    parser = argparse.ArgumentParser(
        description='Export output tables from a SQLite scenario to CSV files.'
    )
    parser.add_argument(
        'source_sqlite',
        help='Path to the source SQLite database file.'
    )
    parser.add_argument(
        'destination_folder',
        help='Path to the destination folder where CSV files will be saved.'
    )
    parser.add_argument(
        '-v', '--version',
        action='version',
        version=f'%(prog)s {script_version}'
    )

    args = parser.parse_args()
    sqlite_db_path = args.source_sqlite
    csv_output_dir = args.destination_folder

    if not os.path.isfile(sqlite_db_path):
        common.logmsg(
            common.error(),
            script_name,
            f"SQLite database file '{sqlite_db_path}' does not exist."
        )
        sys.exit(1)

    if os.path.getsize(sqlite_db_path) == 0:
        common.logmsg(
            common.error(),
            script_name,
            f"SQLite database file '{sqlite_db_path}' is empty."
        )
        sys.exit(1)

    if os.path.exists(csv_output_dir):
        common.logmsg(
            common.error(),
            script_name,
            f"Output directory '{csv_output_dir}' must not already exist."
        )
        sys.exit(1)

    export_failed = common.ompp_tables_to_csv(sqlite_db_path, csv_output_dir)
    if export_failed:
        common.logmsg(
            common.error(),
            script_name,
            "Failed to export tables to CSV."
        )
        sys.exit(1)

    common.logmsg(
        common.info(),
        script_name,
        f"Successfully exported tables from '{sqlite_db_path}' to '{csv_output_dir}'."
    )

if __name__ == "__main__":
    main()
