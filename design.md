# UI

## main page
- overview of servers, and a button to add a new one
- settings page button
- clicking on a server goes to the detail page
- each line that represents a server has:
  - image
  - motd
  - players
  - server jar
  - port
  - start/stop button
  - status icon (stopped, starting, online)

## detail page
`Server` manifest
- container image
- server jar (list from jar `pvc`)
- list of mod jars (list from mod `pvc`)
- memory settings
- idle timeout
- ingress/expose?
- config
  - server-port
  - additional KV
- `pvc` name
- logs (pod stdout)
- send command (pod stdin)
- save button 

## settings page
`OperatorConfig` manifest
- name of server jars `pv`
- name of mod jars `pvc`
- name of servers `pvc`
- save button

# model
## OperatorConfig
settings for operator

## Server
minecraft server

When enabled is set to true, creates a pod with supporting manifests, else they are deleted.
When there are no player for longer than the idle time, enabled is set to false.

## jar pvcs
- have a storage with the server jars
- have a storage with the mod jars

## server pv
used to create pvcs for each server

# Server Pod
## init container
- copies server jar from PVC (to /data/server.jar)
- copies mod jars from PVC (to /data/mods/)
- copies server.properties (to /data/)

## main container
- runs java with memory settings and server.jar