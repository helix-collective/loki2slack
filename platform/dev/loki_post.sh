#!/bin/bash

curl -v -H "Content-Type: application/json" -XPOST -s "http://localhost:3100/loki/api/v1/push" --data-raw \
  '{"streams": [{ "stream": { "env": "devel", "release": "abc", "level": "foobar" }, "values": [ [ "'`date +%s`000000000'", "{\"1\": \"fizz\", \"5\": \"buzz\", \"15\": \"fizzbuzz\"}" ] ] }]}'