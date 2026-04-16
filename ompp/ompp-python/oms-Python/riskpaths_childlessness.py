#
# Python integration example using RiskPaths model
#   to analyze contribution of delayed union formations 
#   versus decreased fertility on childlessness
#
# Input parameters:
#   AgeBaselineForm1: age baseline for first union formation
#   UnionStatusPreg1: relative risks of union status on first pregnancy
# Output value:
#   T05_CohortFertility: Cohort fertility, expression 1
#

# Prerequisite:
#
# download openM++ release from https://github.com/openmpp/main/releases/latest
# unpack it into any directory
# start oms web-service:
#   Windows:
#     cd C:\my-openmpp-release
#     bin\ompp_ui.bat
#   Linux:
#     cd ~/my-openmpp-release
#     bin/oms
#
# Script below is using openM++ web-service "oms"
# to run the model, modify parameters and read output values.

# Important:
#   Script below does not handle errors, please use try/except in production.

import time
import requests
import numpy as np
import matplotlib.pyplot as plt

# get default values for AgeBaselineForm1, UnionStatusPreg1 and SimulationCases parameters 
# by reading it from first model run results
# assuming first run of the model done with default set of parameters
#
rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/run/status/first')
rsp.raise_for_status()
firstRunStatus = rsp.json()
firstRunDigest = rsp.json()['RunDigest']

rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/run/' + firstRunDigest + '/parameter/AgeBaselineForm1/value/start/0/count/0')
rsp.raise_for_status()
ageFirstUnion = rsp.json()

rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/run/' + firstRunDigest + '/parameter/UnionStatusPreg1/value/start/0/count/0')
rsp.raise_for_status()
unionStatusPreg = rsp.json()

rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/run/' + firstRunDigest + '/parameter/SimulationCases/value/start/0/count/0')
rsp.raise_for_status()
simulationCases = rsp.json()

# create new input data for our modelling task
#

# set number of simulation cases
simulationCases[0]['Value'] = 1000

# for AgeBaselineForm1 and UnionStatusPreg1 parameter values
# apply scale in range from 0.44 to 1.0
#
scaleStep = 0.08
scaleValues = [0.44 + i * scaleStep for i in range(1 + round((1.00 - 0.44) / scaleStep))]

ageValues = [x['Value'] for x in ageFirstUnion]
unionValues = [x['Value'] for x in unionStatusPreg]

inpSetLst = []
for scaleAgeBy in scaleValues:
    #
    print("Scale age by:", scaleAgeBy)
    for i in range(len(ageFirstUnion)):
        ageFirstUnion[i]['Value'] = ageValues[i] * scaleAgeBy
    
    for scaleUnionBy in scaleValues:
        #
        # scale first two values of unionStatusPreg vector
        unionStatusPreg[0]['Value'] = unionValues[0] * scaleUnionBy
        unionStatusPreg[1]['Value'] = unionValues[1] * scaleUnionBy
        #
        # create new set of input parameters
        # automatically generate unique names for each input set
        #
        inpSetRq = {
            'ModelName': 'RiskPaths',
            'Name': '',
            'BaseRunDigest': firstRunDigest,
            'IsReadonly': True,
            'Txt': [{
                'LangCode': 'EN',
                'Descr': 'Scale age: ' + str(scaleAgeBy) + ' union status: ' + str(scaleUnionBy) 
            }],
            'Param': [
                {
                    'Name': 'AgeBaselineForm1',
                    'SubCount': 1,
                    'Value': ageFirstUnion,
                    'Txt': [{'LangCode': 'EN', 'Note': 'Age values scale by: ' + str(scaleAgeBy)}]
                },
                {
                    'Name': 'UnionStatusPreg1',
                    'SubCount': 1,
                    'Value': unionStatusPreg,
                    'Txt': [{'LangCode': 'EN', 'Note': 'Union Status values scale by: ' + str(scaleUnionBy)}]
                }
            ],
        }
        #
        # create new input set of model parameters
        # automatically generate unique name for that input set
        #
        rsp = requests.put('http://127.0.0.1:4040/api/workset-create', json=inpSetRq)
        rsp.raise_for_status()
        js = rsp.json()
        #
        inpSetName = js['Name']
        if inpSetName is None or inpSetName == '':
            raise Exception("Fail to create input set, scales:", scaleAgeBy, scaleUnionBy)
        #
        inpSetLst.append(inpSetName)

# create modeling task from all input sets
# automatically generate unique name for the task
#
inpLen = len(inpSetLst)
print("Create task from", inpLen, "input sets of parameters")

taskRq = {
    'ModelName': 'RiskPaths',
    'Name': '',
    'Set': inpSetLst,
    'Txt': [{
        'LangCode': 'EN',
        'Descr': 'Task to run RiskPaths ' + str(inpLen) + ' times',
        'Note': 'Task scales AgeBaselineForm1 and UnionStatusPreg1 parameters from 0.44 to 1.00 with step ' + str(scaleStep)
    }]
}
rsp = requests.put('http://127.0.0.1:4040/api/task-new', json=taskRq)
rsp.raise_for_status()
js = rsp.json()

taskName = js['Name']
if taskName is None or taskName == '':
    raise Exception("Error at create modeling task")

#
# submit request to web-service to run RiskPaths with modeling task
#
runModelRq = {
    'ModelName': 'RiskPaths',
    'Opts': {
        'OpenM.TaskName': taskName,
        'OpenM.ProgressPercent': '100'
    }
}
rsp = requests.post('http://127.0.0.1:4040/api/run', json=runModelRq)
rsp.raise_for_status()
js = rsp.json()
#
taskRunStamp = js['RunStamp']
if taskRunStamp is None or taskRunStamp == '':
    raise Exception('Model failed to start, task run stamp is empty')

print("Starting modeling task:", taskName)

# wait until modeling task completed
# and report the progress
#
# task status returned by web-service can be one of:
# i=initial p=in progress w=waiting s=success x=exit e=error(failed)
#
taskStatus = '' 

while taskStatus in '' 'i' 'p' 'w':
    #
    time.sleep(1)
    #
    rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/task/' + taskName + '/run-status/run/' + taskRunStamp)
    rsp.raise_for_status()
    js = rsp.json()
    taskStatus = js['Status']
    #
    # if model not started to run the task yet check again after short sleep
    #
    if taskStatus in '' 'i':
        #
        print("Waiting for modeling task to start...")
        continue
    #
    # if task completed successfully then get pairs of {model run, inpur set name}
    #
    if taskStatus == 's':
        rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/task/' + taskName + '/runs')
        rsp.raise_for_status()
        js = rsp.json()
        taskRuns = js['TaskRun'][0]['TaskRunSet']   # use index=0 because this is first run of our task
        break
    #
    # if task still in progress then count completed model runs
    #
    if taskStatus in 'i' 'p' 'w':
        rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/run/' + taskRunStamp + '/status/list')
        rsp.raise_for_status()
        trs = rsp.json()
        #
        n = 0
        for r in trs:
            if r['Status'] == 's': n += 1
        #
        print("Completed", n, "model runs out of", inpLen)
        continue
    #
    # any other task run status considered as failure
    #
    raise Exception("Model run failed, task run stamp:", taskRunStamp, "status:", taskStatus)
    #

print("Modeling task completed, retriving results...")

# for each age and union status retrive output:
#   childlessness value: T05_CohortFertility.Expr1
#
# organize results into 2-dimensional array to plot 3d chart
#
childlessnessVals = np.zeros((len(scaleValues), len(scaleValues)))
runIdx = 0

for ageIdx in range(len(scaleValues)):
    for unionIdx in range(len(scaleValues)):
        #
        runDigest = taskRuns[runIdx]['Run']['RunDigest']
        #
        rsp = requests.get('http://127.0.0.1:4040/api/model/RiskPaths/run/' + runDigest + '/table/T05_CohortFertility/expr')
        rsp.raise_for_status()
        js = rsp.json()
        #
        childlessnessVals[ageIdx][unionIdx] = js[1]['Value']
        runIdx += 1

# display the results
#
ageVals, unionVals = np.meshgrid(scaleValues, scaleValues)

fig = plt.figure()
ax = plt.axes(projection='3d')
ax.plot_wireframe(ageVals, unionVals, childlessnessVals, color='black')
ax.set_xlabel('Decreased union formation')
ax.set_ylabel('Decreased fertility')
ax.set_zlabel('Childlessness')
ax.view_init(elev=45)
plt.show()

