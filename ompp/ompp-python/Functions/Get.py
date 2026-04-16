"""
Author: Scott Yun Ho
Date: Oct. 29, 2024
Description: Functions that get unspecific information from OpenM++.
"""

import requests
import pandas as pd


"""
Description: Provided that the OpenM++ web service (the "oms") is running at oms_url, get list of models from the API route oms_url/api/model-list.
Parameters: 
Return: 
"""
def get_models(oms_url) :
    x = requests.get(oms_url + '/api/model-list').json()
    return pd.DataFrame([item['Model'] for item in x])


"""
Description: Provided that the OpenM++ web service (the "oms") is running at oms_url, get full list of model runs from API route at oms_url/api/model/model_digest/run-list.
Parameters: 
Return: 
"""
def get_model_runs(oms_url, model_digest):
    x = requests.get(oms_url + '/api/model/' + model_digest + '/run-list').json()
    return pd.DataFrame([item for item in x])

"""
Description: 
Parameters: 
Return: 
"""
def get_scenarios():
    return 1