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

Configure [Local Grafana](http://localhost:3000/?orgId=1) with `loki` as a data source named `Loki`.

Notes:
- The slack token must have `chat:write` scope.
  - So far only managed this with user token (bot token should work).
- For links in the eventual slack message to open in the correct place the data source name must match in the `loki2slack.cfg` file.

## Post to Slack Directly

Can be useful to shortcut things, remove Loki for the equation and post to Slack directly.
To do this use `loki2slack post -c <config_file> --sample-file <example>`.
The example file must have a least two line.
The first is the Loki link and the second the log line.
These can be see when running tail with debug (`loki2slack tail --debug ...`).

**TODO**
- include an example.
- make this easier.

## Build & Release

### Using Multipart Docker File
``` bash
docker build --file Dockerfile.multipart --tag helixta/loki2slack:latest .
docker tag helixta/loki2slack:latest helixta/loki2slack:`git describe`
docker push helixta/loki2slack:latest
docker push helixta/loki2slack:`git describe`
```

### Using `goreleaser`

Export a `GITHUB_TOKEN` as an env var.
```
goreleaser
```

## Deploy

**Note** ensure Loki is configure for grpc. IE
```
server:
  http_listen_port: 3100
  grpc_listen_port: 9096
...
```

Entries in the `loki2slack.cfg` file much match your deployment setup.

## CI

TODO:
- goreleaser
- github actions
