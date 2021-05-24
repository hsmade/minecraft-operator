# Minecraft Operator
This operator assists in running multiple Minecraft servers.
I wrote this, as my daughter comes up with a lot of different world ideas,
and wants to keep them all available to play on. As it's not feasible to run them all on my server at home,
and I don't want to have to start them on (her) demand, I wrote a simple web-app that lets her manage them.
Now to overcome the last 'burden' of manually having to create the directories, scripts, config, etc, every time
that she wants a new server, I started working on this operator.

This operator consumes the `Server` CRD, which lets you define the specifics about the server you want to run.
It also has a web UI that allows you to enable/disable the Servers. You can configure an idle timeout on the Server
object, to let it shut down after the last player left, and the said timeout has expired.

## Running the operator

## Example Server definition

## Development
### TODO
See the TODO/FIXME annotations in the code.
Also:

 - refactor controller
 - write tests
 - fix thumbnails
 - fix ui ordering
 - implement ingress as alternative to hostPort

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