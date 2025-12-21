#!/bin/sh

# start a proxy server
## go run ./file/proxy  &

http --proxy http:http://localhost:8080  GET https://jsonplaceholder.typicode.com/todos/1

curl -x http://localhost:8080 https://jsonplaceholder.typicode.com/todos/1


cat /tmp/proxy.log