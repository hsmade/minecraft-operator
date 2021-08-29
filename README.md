# Minecraft Operator
This operator assists in running multiple Minecraft servers.
I wrote this, as my daughter comes up with a lot of different world ideas,
and wants to keep them all available to play on. As it's not feasible to run them all on my server at home,
and I don't want to have to start them on (her) demand, I wrote a simple web-app that lets her manage them.
Now to overcome the last 'burden' of manually having to create the directories, scripts, config, etc, every time
that she wants a new server, I started working on this operator.

### Server
This operator consumes the `Server` CRD, which lets you define the specifics about the server you want to run.
It also has a web UI that allows you to enable/disable the Servers. You can configure an idle timeout on the Server
object, to let it shut down after the last player left, and the said timeout has expired.

### Mod
The `Mod` CRD specifies a mod, its version and the URL to download it from. 
These are referenced from the `Server` manifest in the `Mods` list.

## Running the operator

## Example Server definition

## Development
### run locally
```bash
make install manifests generate fmt vet && go run ./main.go --zap-log-level 9
```
### deploy
```bash
make docker-build docker-push deploy
```

### TODO
See the TODO/FIXME annotations in the code.
Also:

 - write tests
 - implement ingress as alternative to hostPort
 - fix forge, needs additional files?
 - have some visual feedback on starting

### Log levels
This project uses [logr](https://github.com/go-logr/logr), which has no log levels like `Debug` or `Warning`.
Instead, it has `Info`, `Error` and numerical levels.
This project uses the following levels:

 - `0/Info`: stuff that the user should be able to see in the logs, always
 - `2/Error`: stuff that broke, that we need to tell the user in the logs, always
 - `5/Verbose`: stuff that is sometimes handy to know.
 - `7/Flow`: program flow   
 - `9/Trace`: program flow with full details, for debugging nasty errors

The first two levels are compatible with / implemented by logr.

## Adding new server JARs
Forge 1.17:
```bash
$ cd /srv/minecraft/jars/server
$ mkdir forge-1.17.1
$ cd forge-1.17.1
$ java -jar forge-1.17.1-37.0.45-installer.jar --installServer
$ cat << EOF > start.sh
#!/bin/bash
java -Xmx ${XMX:-1024M} -Xms ${XMS:-1024M} @libraries/net/minecraftforge/forge/1.17.1-37.0.45/unix_args.txt nogui
EOF
$ chmod +x start.sh
```

Forge < 1.17:
```bash
$ cd /srv/minecraft/jars/server
$ mkdir forge-1.16.5
$ cd forge-1.16.5
$ java -jar forge-1.16.5-36.2.2-installer.jar --installServer
$ cat << EOF > start.sh
#!/bin/bash
exec java -Xmx ${XMX:-1024M} -Xms ${XMS:-1024M} -jar forge-1.16.5-36.2.2.jar nogui
EOF
$ chmod +x start.sh
```

Vanilla:
```bash
$ cd /srv/minecraft/jars/server
$ mkdir vanilla-1.16.5
$ cd vanilla-1.16.5
$ wget https://launcher.mojang.com/v1/objects/1b557e7b033b583cd9f66746b7a9ab1ec1673ced/server.jar
$ cat << EOF > start.sh
#!/bin/bash
exec java -Xmx ${XMX:-1024M} -Xms ${XMS:-1024M} -jar server.jar nogui
EOF
$ chmod +x start.sh
```