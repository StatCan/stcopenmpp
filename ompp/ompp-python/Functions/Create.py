#To-do: Finished logic but adjust code to work as a module
"""
Author: Scott Yun Ho
Date: Oct. 29, 2024
Description: Functions that create something new on the server-side.
"""

import requests
import pandas as pd
import time


"""
Description: Create and run a given model on OpenM++ engine. Send a request to run model, then once finished, return model_run_status.
Parameters: 
Return: 
"""

def create_model_run(model_digest, oms_url, simulation_cases=1e5, tables=[], threads=1, sub_samples=1) :

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
      
        # Pause for one second to give time for information to be delivered. 
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

    return model_run_status

"""
Description: 
Parameters: 
Return: 
"""
def create_scenario():
    return 1