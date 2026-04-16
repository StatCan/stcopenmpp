Azure example of small cluster configuration files:
- front-end server with 4 cores, host name: dm
- 2 back-end servers, 4 cores each, host names: dc1, dc2
- Azure resource group name: dm_group

Directory oms contains init.d scripts to start 2 instances of oms from /mirror/data/oms.
- "demo" oms instance at http://localhost:4044
- "test" oms instance at http://localhost:4042

