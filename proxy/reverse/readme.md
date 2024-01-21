# Running example

## Memcached server

Launch `memcached` with the following command:

```bash
docker run --name memcached --publish 11211:11211 memcached:latest -m 64
```

## Start the web server on port 8081

```bash
~/.local/bin/http_server.py -p 8081
```

Use the web server in the gist: https://gist.github.com/ricleal/72efc72d7de5e23ce98b9afb2973232a

## Start the proxy server on port 8080

```bash
go run ./proxy/reverse
```

## Test the proxy server

```bash

http -v localhost:8080/x

http -v localhost:8080/x

http -v POST localhost:8080/x <<< 'foo bar'

http -v POST localhost:8080/x <<< 'foo bar'
```