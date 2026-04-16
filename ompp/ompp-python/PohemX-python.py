"""
Author: Scott Yun Ho
Date: Sept. 17, 2024
Description: Python script to run PohemX model to return output tables specified in config file.
"""
# Note: "model-digest" basically means model ID.
# To-do: maybe add in an initial API call then dump all of the info out because sometimes on initial run, the API call will finish "successfully" in 3 seconds, but return an empty list that causes an error later. This can be solved by just running the model again, but maybe consider making this running the model again part an actual portion of the code to circumvent this entirely.
# To-do: mimic R folder from Matt's repo, not just the main functions.

# Dependencies
import requests
import subprocess  # Opens web service (oms) and creates subprocesses to run program. 
import time
import os  # Necessary to create folder in current directory to load model outputs into.
import csv  # Necessary to decode CSV download for model output table results.
import json  # Necessary to load list from config.ini.
import time

import pandas as pd
import openpyxl  # Necessary to use pandas to_excel method.
import configparser  # Necessary to load and process config.ini file.


# configparser setup informed by: https://www.geeksforgeeks.org/how-to-write-a-configuration-file-in-python/
config = configparser.ConfigParser()
config.read('.\config.ini')

# Access values from the configuration file. 
model_name = config.get('Run Settings', 'model_name')
oms_url = config.get('Run Settings', 'oms_url')
tables = json.loads(config.get('Run Settings', 'tables'))
simulation_cases = config.getfloat('Run Settings', 'simulation_cases')
threads = config.getint('Run Settings', 'threads')
sub_samples = config.getint('Run Settings', 'sub_samples')
path_to_ompp = config.get('Paths', 'path_to_ompp')
outputs_folder = config.get('Paths', 'outputs_folder')
#expr_name_constant = config.get('Miscellaneous', 'expr_name_constant')
#expr_value_constant = config.get('Miscellaneous', 'expr_value_constant')


# Load user-defined functions
def get_models() :
    x = requests.get(oms_url + '/api/model-list').json()
    return pd.DataFrame([item['Model'] for item in x])

def get_model_digest(df, model_name):
    return df['Digest'][df.loc[df['Name'] == model_name].index.values[0]]

def get_model_runs(model_digest):
    x = requests.get(oms_url + '/api/model/' + model_digest + '/run-list').json()
    return pd.DataFrame([item for item in x])

def get_output_table(model_run_digest, table):
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

def run_model(model_digest, simulation_cases = 1e5, tables = [], threads = 1, sub_samples = 1) :
    
    # Coerce numeric parameters to strings
    simulation_cases = str(int(simulation_cases))
    threads = str(int(threads))
    sub_samples = str(int(sub_samples))
  
    # Set JSON request
    json_request = {
        'ModelName': model_digest,
        'Opts': {
            'Parameter.SimulationCases': simulation_cases,
            'OpenM.Threads': threads,
            'OpenM.SubValues': sub_samples,
            'OpenM.LogToConsole': 'true',
            'OpenM.ProgressPercent': '100'
        },
        'Tables': tables
    }
  
    # Send request to oms
    print("Sending API request to web service...")
    x = requests.post(oms_url + '/api/run', json = json_request)
  
    # Set model run status
    model_run_status = ''
  
    startTime = time.time() #have start time, then have currentTime in model progress to repeatedly check how much time has elapsed.
    # Monitor model run status
    while model_run_status in '' 'i' 'p' 'w' :
      
        # Pause for one second
        time.sleep(1)
    
        # Get current model run status
        model_run_status = requests.get(oms_url + '/api/model/' + model_digest + '/run/status/last').json()['Status']
        currentTime = time.time()

        # Render conditional message to console 
        if model_run_status in '' 'i' :
            print("Waiting for model run to start. ", end="")
            print( "Current time elapsed:", round((currentTime - startTime), 2), "seconds so far... ")
            continue

        elif model_run_status == 's' :
            print("Model run completed successfully! ", end="")
            print("Final time elapsed:", round((currentTime - startTime), 2), "seconds. \n")
            break

        elif model_run_status in 'i' 'p' 'w' :
            print("Model run in progress. ", end="")
            print("Current time elapsed:", round((currentTime - startTime), 2), "seconds so far... ")
            continue

        elif model_run_status == 'e' :
            print("Model run completed, but with errors. :(")
            print("Final time elapsed:", round((currentTime - startTime), 2), "seconds. \n")
            break

    # Create model run object
    model_run = {
        'model': model_name, 
        'digest': get_model_runs(model_digest).iloc[-1]['RunDigest'], 
        'status': model_run_status
    }

    # Retrieve specified output tables after model run.
    for table in tables:
        print('Retrieving ' + table + ' table...')
        model_run[table] = get_output_table(model_run['digest'], table)
    return model_run

def download_csv(file_name, df_table):
    return df_table.to_csv(file_name, index=True)

def download_excel(file_name, df_table):
    return df_table.to_excel(file_name)

# Create folder within current directory and switch to directory for output download.
def create_output_folder(output_folder_name=outputs_folder):
    current_dir = os.getcwd()
    folder_destination = os.path.join(current_dir, output_folder_name)
    if not os.path.exists(folder_destination):
        os.makedirs(folder_destination)
    os.chdir(output_folder_name)


try: 
    # Start OpenM++ web service (oms)
    print("\n\nStarting OpenM++ web service as new process. Web service window will open shortly...\n")
    oms = subprocess.Popen(
      path_to_ompp + '/bin/start_oms.sh',  # Can do either /bin/ompp_ui.bat or /bin/start_oms.sh. bat file will also open localhost:4040 in browser in addition to starting oms.exe.
      shell = True, 
      stdout = subprocess.PIPE,
      creationflags = subprocess.CREATE_NEW_PROCESS_GROUP  # https://stackoverflow.com/questions/47016723/windows-equivalent-for-spawning-and-killing-separate-process-group-in-python-3#:~:text=5-,THIS%20ANSWER,-IS%20PROVIDED%20BY
    )  # creationflags process group makes killing the processes at the end of the program much cleaner.

    time.sleep(2)

    # Get models
    print("Retrieving list of models from OpenM++...")
    models = get_models()

    # Get model digest (unique identifier)
    print("Retrieving model digest (unique ID code)...")
    model_digest = get_model_digest(models, model_name)

    # Get model runs
    print("Retrieving list of model runs to get initial run digest value...\n")
    model_runs = get_model_runs(model_digest)

    # Get base run digest
    base_run_digest = model_runs['RunDigest'][0]

    # Run a model
    print("Starting model run...")
    model_run = run_model(
        model_digest,
        simulation_cases,
        tables,
        threads,
        sub_samples
    )

    # Get model run digest
    model_run['digest']

    # Get model run status
    # See https://github.com/openmpp/openmpp.github.io/wiki/Oms-web-service for model run status codes
    model_run['status']

    # Create outputs folder in current directory.
    print("Creating output folder in current directory... ")
    create_output_folder()

    """
    # Get and download first output table
    output_table_1 = model_run[tables[0]]
    print(output_table_1)    
    print("Downloading " + tables[0] + " in CSV format.")
    download_csv(tables[0] + ".csv", output_table_1)
    print("Downloading " + tables[0] + " in Excel format.")
    download_excel(tables[0] + ".xlsx", output_table_1)

    # Get and download second output table
    output_table_2 = model_run[tables[1]]
    print(output_table_2)
    print("Downloading " + tables[1] + " in CSV format.")
    download_csv(tables[1] + ".csv", output_table_2)
    print("Downloading " + tables[1] + " in Excel format.")
    download_excel(tables[1] + ".xlsx", output_table_2)
    """

    # Get and download output tables specified in tables list.
    for table in tables:
        output_table = model_run[table]
        print(output_table)
        #print(table)
        print("Downloading " + table + " in CSV format.")
        download_csv(table + ".csv", output_table)
        print("Downloading " + table + " in Excel format.")
        download_excel(table + ".xlsx", output_table)
    print("\n")

except Exception as e:
    print("Error while getting and downloading output tables.")
    print(e)

finally:
    # Processes cleanup informed by: https://stackoverflow.com/questions/4789837/how-to-terminate-a-python-subprocess-launched-with-shell-true#:~:text=Popen(%22TASKKILL%20/F%20/IM%20%22%20%2B%20process_name)
    print("Program stopping. Killing OpenM++ Web Service. ")
    subprocess.Popen("TASKKILL /F /PID " + str(oms.pid) + " /T")
    subprocess.Popen("TASKKILL /F /IM " + "oms.exe")
    subprocess.Popen("TASKKILL /F /IM " + "bash.exe")
    print("Program finished. Tab will close in 30 seconds. ")
    time.sleep(30)