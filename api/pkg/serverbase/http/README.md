# HTTP server base
This package aims to provide a set of functionalities to facilitate the creation of http-based servers. The server provides a logger with default good configuration that the user can leverage.

The user must:

1) call `NewHttpServerBase` to create a new base server
2) define an object that satisfies the interface `Server`
3) call `base.Serve` with the given object

This will create an HTTP server that has the following endpoints by default:
* `/status/live`, which is an HTTP wrapper around the `Live` method of the interface
* `/status/ready`, which is an HTTP wrapper around the `Ready` method of the interface
* `/status/metrics`, which is a prometheus-compatible endpoint exposing the metrics for the service and requests
* `/debug/enable` and `/debug/disable`, which is a POST endpoint used to dynamically control the log level in the provided logger
