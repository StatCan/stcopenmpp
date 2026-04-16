"""
Author: Scott Yun Ho
Date: Oct. 17, 2024
Description: 
"""

# To-do: Create sample run folder with different use cases in each file. E.g. 1) straightforward model run, 2) test of each individual function with test cases.
# To-do: Figure out why infinity symbol is in output even though the option is set properly.
# To-do: rename this file and PoheMX-python into sample-model-run-test.py and sample-functions-test.py

# Packages you don't need to manually install. 
import subprocess  # Opens web service (oms) and creates subprocesses to run program. 
import time
import json
import configparser

# Key functions to import from local files.
import Functions.Get as Get
import Functions.Load as Load
import Functions.Create as Create
import Functions.Delete as Delete
import Functions.Utilities as Utilities


# Retrieve constants from .\config.ini file. Setup informed by: https://www.geeksforgeeks.org/how-to-write-a-configuration-file-in-python/
config = configparser.ConfigParser()
config.read('.\config.ini')

config_model_name = config.get('Run Settings', 'model_name')
config_oms_url = config.get('Run Settings', 'oms_url')
config_tables = json.loads(config.get('Run Settings', 'tables'))
config_simulation_cases = config.get('Run Settings', 'simulation_cases')
config_threads = config.get('Run Settings', 'threads')
config_sub_samples = config.get('Run Settings', 'sub_samples')
config_path_to_ompp = config.get('Paths', 'path_to_ompp')
config_outputs_folder = config.get('Paths', 'outputs_folder')



try: 
    # Start OpenM++ web service (oms)
    print("\n\nStarting OpenM++ web service as new process. Web service window will open shortly...\n")
    oms = subprocess.Popen(
        config_path_to_ompp + '/bin/start_oms.sh',
        shell = True, 
        stdout = subprocess.PIPE,
        creationflags = subprocess.CREATE_NEW_PROCESS_GROUP  # https://stackoverflow.com/questions/47016723/windows-equivalent-for-spawning-and-killing-separate-process-group-in-python-3#:~:text=5-,THIS%20ANSWER,-IS%20PROVIDED%20BY
    )  # creationflags process group makes killing the processes at the end of the program much cleaner.

    time.sleep(2)

    # SAMPLE - Run get_models() from ./Functions/Get.py.
    print("\nList of Models:")
    model_list = Get.get_models(config_oms_url)
    print(model_list)
    print("\n")

    # SAMPLE - Run get_scenario() from ./Functions/Get.py.

    # SAMPLE - Run get_model_runs() from ./Functions/Get.py.
    print("List of Model Runs:")
    retrieved_model_digest = Load.load_model_digest(model_list, config_model_name)
    model_runs = Get.get_model_runs(config_oms_url, retrieved_model_digest)
    print(model_runs)
    print("\n")

    # SAMPLE - Run create_scenario() from ./Functions/Create.py.


    # SAMPLE - Run create_model_run() from ./Functions/Create.py.
    print("Run Model Sample:")
    model_run_status = Create.create_model_run(model_digest=retrieved_model_digest, oms_url=config_oms_url, simulation_cases=float(config_simulation_cases), tables=config_tables, threads=config_threads, sub_samples=config_sub_samples)
    print(model_run_status)
    print("\n")

    # SAMPLE - Run load_model() from ./Functions/Load.py.
    print("Load Model Example:")
    model_run = Load.load_model_run(model_name=config_model_name, model_digest=retrieved_model_digest, model_run_status=model_run_status, oms_url=config_oms_url, tables=config_tables)
    print(model_run)
    print("\n")


    # SAMPLE - Download output files. Run create_output_folder(), download_csv(), and download_excel() from ./Functions/PyOpenMPP_Utilities.py.
    Utilities.generate_output_files(model_run=model_run, tables=config_tables, output_folder_name=config_outputs_folder)


    # SAMPLE - Run load_scenario() from ./Functions/Load.py.
    # SAMPLE - Run load_model_run() from ./Functions/Load.py.
    # SAMPLE - Run load_model_runs() from ./Functions/Load.py.
    # SAMPLE - Run load_output_tables() from ./Functions/Load.py.
    # SAMPLE - Run load_model_digest() from ./Functions/Load.py.


    # SAMPLE - Run delete_scenario() from ./Functions/Delete.py.
    # SAMPLE - Run delete_model_run() from ./Functions/Delete.py.

    # SAMPLE - Utilities

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
    time.sleep(5)