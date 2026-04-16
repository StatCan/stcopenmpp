"""
Author: Scott Yun Ho
Date: Oct. 29, 2024
Description: Helper functions that don't directly perform an API call or don't interact with OpenM++ in general.
"""

import pandas as pd
import os



"""
Description: 
Parameters: 
Return: 
"""
def download_csv(file_name, df_table):
    return df_table.to_csv(file_name, index=True)

"""
Description: 
Parameters: 
Return: 
"""
def download_excel(file_name, df_table):
    return df_table.to_excel(file_name)

# Create folder within current directory and switch to directory for output download.
"""
Description: 
Parameters: 
Return: 
"""
def create_output_folder(output_folder_name):
    current_dir = os.getcwd()
    folder_destination = os.path.join(current_dir, output_folder_name)
    if not os.path.exists(folder_destination):
        os.makedirs(folder_destination)
    os.chdir(output_folder_name)

"""
Description: 
Parameters: 
Return: 
"""
def generate_output_files(model_run, tables, output_folder_name):
    # Create output folder.
    create_output_folder(output_folder_name)

    # Get and download output tables specified in tables list.
    for table in tables:
        output_table = model_run[table]
        #print(table)
        print("Downloading " + table + " in CSV format.")
        download_csv(table + ".csv", output_table)
        print("Downloading " + table + " in Excel format.")
        download_excel(table + ".xlsx", output_table)
    print("\n")