"""
Author: Scott Yun Ho
Date: Oct. 29, 2024
Description: Functions that get specific information from OpenM++.
"""
import Functions.Get as Get

import requests
import pandas as pd
import csv

"""
Description: Retrieve specific model run based on model_name and model_digest. Requires model_run_status from Create_model_run.py.
Parameters: 
Return: 
"""
def load_model_run(model_name, model_digest, model_run_status, oms_url, tables):

    # Create model run object. Run has now been completed, so create an object that gathers the finished information in one place. 
    model_run = {
        'model': model_name, 
        'digest': Get.get_model_runs(oms_url, model_digest).iloc[-1]['RunDigest'], 
        'status': model_run_status
    }

    # Retrieve specified output tables after model run.
    print(tables)
    for table in tables:
        print('Retrieving ' + table + ' table...')
        model_run[table] = load_output_table(model_digest=model_digest, model_run_digest=model_run['digest'], table=table, oms_url=oms_url)

    return model_run

#To-do: finished logic here but test
"""
Description: 
Parameters: 
Return: 
"""
def load_model(model_name):
    x = requests.get(oms_url + '/api/model-list').json()
    model_list = pd.DataFrame([item['Model'] for item in x])
    return model_list.loc[model_list['ModelName'] == model_name]

#To-do: get multiple specific runs. Take array as input and output info for multiple model runs.
"""
Description: 
Parameters: 
Return: 
"""
def load_model_runs():
    return 1

"""
Description: 
Parameters: 
Return: 
"""
def load_output_table(model_digest, model_run_digest, table, oms_url):
    # Get list from CSV format (data is neater). URL returns CSV file attachment in a response stream, which is why stream is necessary below. Informed by: https://stackoverflow.com/questions/35371043/use-python-requests-to-download-csv#:~:text=CSV_URL%20%3D%20%27http%3A//samplecsvs.s3.amazonaws.com/Sacramentorealestatetransactions.csv%27-,with%20requests,-.Session()%20as%20s%3A%0A%20%20%20%20download%20%3D%20s.get(CSV_URL)%0A%0A%20%20%20%20decoded_content
    print("Getting response stream output and processing into a dataframe...")
    raw_csv_data = []
    with requests.Session() as stream:
        download = stream.get(oms_url + '/api/model/' + model_digest + '/run/' + model_run_digest + '/table/' + table + '/expr/csv')
        decoded_content = download.content.decode('utf-8')
        cr = csv.reader(decoded_content.splitlines(), delimiter=',')
        raw_csv_data = list(cr)

    # Convert list of lists table output (raw_csv_data) to pandas dataframe.
    headers = raw_csv_data.pop(0)
    long_format_df = pd.DataFrame(raw_csv_data, columns=headers)

    # Clean data to remove negative and positive infinity. regex=True needed to recognize the unicode for infinity: \u221e
    #print("Cleaning infinity symbols from data...")
    #long_format_df = long_format_df.replace("-"+"\u221e", "MIN", regex=True) 
    #long_format_df = long_format_df.replace("\u221e", "MAX", regex=True)

    # Previous format that returned everything in dims col.
    """
    #response = pd.DataFrame(requests.get(oms_url + '/api/model/' + model_digest + '/run/' + model_run_digest + '/table/' + table + '/expr').json())
    #print(pd.DataFrame(requests.get(oms_url + '/api/model/' + model_digest + '/run/' + model_run_digest + '/table/' + table + '/expr/csv')))
    #response = requests.get(oms_url + '/api/model/' + model_digest + '/run/' + model_run_digest + '/table/' + table + '/expr/csv')
    #print(response.headers)
    #output_df = pd.DataFrame(response.content)
    #print(response)
    #output_df = response

    #output_df["Dims"] = output_df["Dims"].astype(str)  # Program cannot pivot using a list as index, so Dims col must be converted to string first.
    #output_df = output_df.pivot(index="Dims", columns="ExprId", values="Value")
    #print(output_df)
    """

    # Copy headers to use as columns during pivot, but not including expr_name and expr_value since we're pivoting on these values.
    wide_format_indices = headers
    wide_format_indices.remove("expr_name")
    wide_format_indices.remove("expr_value")

    # Pivot dataframe to be wide format.
    print("Converting data to wide format...\n")
    wide_format_df = long_format_df
    wide_format_df = pd.pivot(wide_format_df, 
        index=wide_format_indices, 
        columns="expr_name", 
        values="expr_value"
    )

    return wide_format_df

"""
Description: 
Parameters: 
Return: 
"""
def load_scenario():
    return 1


def load_model_digest(df, model_name) :
    return df['Digest'][df.loc[df['Name'] == model_name].index.values[0]]
