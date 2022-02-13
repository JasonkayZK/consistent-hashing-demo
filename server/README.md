# Server

A simple Server to handle k-v cache for testing.

## How to use

Before start the server, you need to start the proxy server first:

```shell
$ go run main.go
```

This will start a proxy server on port 18888.

Then you can start the server:

```shell
$ go run server/main.go
```

This will start a server on port 8080 in default.

You can also specify the port:

```shell
$ go run server/main.go -p 8081
```

The server will listen on port 8081.
