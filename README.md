# loki2slack

A Loki to Slack message forwarder.

## Quick Start

```
cd platform/dev
docker-compose --project-name example_logging up -d
docker-compose --project-name example_logging logs -f
```
Now is a separate terminal
```
./platform/dev/loki_post.sh
```

To post to Slack, configure the `platform/dev/loki2slack.cfg` with `SlackToken` and `SlackChannelId`, restart `loki2slack` container.
Note the slack token must have `chat:write` scope.
So far only managed this with user token (bot token should work).

## Build

``` bash
docker build --tag helixta/loki2slack:latest .
docker tag helixta/loki2slack:latest helixta/loki2slack:`git describe`
```
## Release

``` bash
docker push helixta/loki2slack:latest
docker push helixta/loki2slack:`git describe`
```

## Deploy

### Via Docker Compose

``` yaml
services:

  loki2slack:
    image: helixta/loki2slack
    volumes:
      - ./config/loki2slack.cfg:/config/loki2slack.cfg
    command: --logtostderr tail -c /config/loki2slack.cfg

  loki:
    image: grafana/loki:2.2.1
    ports:
      - "3100:3100"
      - "9096:9096"
    volumes:
      - ./config/loki-config.yaml:/etc/loki/local-config.yaml
    command: -config.file=/etc/loki/local-config.yaml

  grafana:
    image: grafana/grafana:8.0.4
    ports:
      - "3000:3000"
```

**Note** ensure Loki is configure for grpc. IE
```
server:
  http_listen_port: 3100
  grpc_listen_port: 9096
...
```

Edit the `loki2slack.cfg` file.

## CI

TODO
