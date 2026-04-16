# **PyOpenMPP**

Python implementation of OpenM++ via API calls to web service. Currently is tested primarily with PohemX model. 

Run by specifying options in config.ini file at root, then running PohemX-python.py. This can be built into an executable which can also work off of the config.ini file. 

Functions folder modularizes the different API calls to OpenM++ web service. These are applied and tested in PyOpenMPP_main.py.


## <b>Modules</b>

### *<ins>Create Model Run</ins> - Functions/Create.py/create_model_run()*
#### Description:
This method takes information and then creates/executes a model run in your running instance of OpenM++ (via the oms process; the OpenM++ Web Service).
#### API Calls:
- [ ] Submit model run job: **POST** ```<oms_url>/api/run```
- [ ] Retrieve status of model run: **GET** ```<oms_url>/api/model/<model_digest>/run/status/last```
#### Parameters: 
- [ ] ```model_digest``` specifies what model to run since model_digest is basically a unique ID that each model has. For example, if you were running PohemX, model_digest would correspond to PohemX specifically.
- [ ] ```simulation_cases``` specifies the number of 'actors' to be simulated. The create_model_run function will set this to 1e5 by default, but will be overwritten by whatever is set in the config.ini file at runtime.
- [ ] ```tables``` specifies which output tables to return in the output. The create_model_run function will set this to an empty list by default, but will be overwritten by whatever is set in the config.ini file at runtime.
- [ ] ```threads``` specifies how many threads to run this model with. The create_model_run function will set this to 1 by default, but will be overwritten by whatever is set in the config.ini file at runtime. 
- [ ] ```sub_samples``` specifies how many samples of the start population to run, which then gets aggregated at the end. Essentially, it specifies how many times you want the model to run. More times will eat up more memory and space but return a more accurate simulation. 
#### Output: 
Returns ```model_run_status``` variable, which specifies the state of the model run after it finishes running server-side. 

---
### *<ins>Load Model Run</ins> - Functions/Load.py/load_model_run()*
#### Description: 
Retrieve specific model run based on model_name and model_digest. Requires model_run_status from Create_model_run.py.
#### API Calls:
API calls done via get_model_runs method and load_output_table method.
#### Parameters: 
#### Output: 
Returns model run object with all info about a finished model run (presumably) packed together nicely.
To-do: create a get_model_run_status function in case user wants to load a previous model run where you don't have the status because it was run a long time ago. 

---
### *<ins>Get Models</ins> - Functions/Get.py/get_models()*
#### Description:
This module retrieves all models accessible in your running instance of OpenM++.
#### API Calls:
- [ ] Retrieve list of models: **GET** ```<oms_url>/api/model-list```
#### Parameters:
- [ ] ```oms_url``` specifies the URL that your running instance of OpenM++ is sending information to. For example, ```http://localhost:4040```.
#### Output:
Returns a Pandas dataframe of models list.

---

### *<ins>Load Output Table</ins> - Functions/Load.py/load_output_table()*
#### Description: 
#### Parameters: 
#### Output: 

---






## <b>Building Executable File</b>
Use pyinstaller to build PohemX-python.py into an executable file.

```pyinstaller --onefile PohemX-python.py --hidden-import openpyxl --hidden-import configparser```

- [ ] ```--onefile``` packages the dependencies in with the executable. 
- [ ] ```PohemX-python.py``` specifies the Python file to create an executable version of. 
- [ ] ```--hidden-import``` specifies to package a dependency that isn't already in pyinstaller's list of recognized imports. This is done for openpyxl and configparser.

**Note:** The executable file will only work if config.ini is in the same directory, and if you update your OpenM++ path in config.ini.


## <b>Creating a Virtual Environment</b>
Necessary if you want to cleanly contain this project's Python dependencies during development. Instructions pulled from [here](https://python.land/virtual-environments/virtualenv). 

venv is included in Python by default, so go to command line`or PowerShell, and cd to where you'd like to create your virtual environment directory. Then, enter the following:
```python -m venv <directory>```

Once created, cd into the directory you've created, and enter the following:
```
# In cmd.exe
venv\Scripts\activate.bat
# In PowerShell
venv\Scripts\Activate.ps1
```

Now you've successfully created your virtual environment. You can install packages while this is activated so that these packages won't messily take up space outside of this directory. 

If you'd like to exit your virtual environment, enter the following:
```deactivate```

**Note:** Once you activate your virtual environment, you must turn it on again in any window you use to run your program, or in any window you use to install more packages (assuming these are packages you want to be in this virtual environment). 
