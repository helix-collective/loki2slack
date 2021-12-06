# Post to Slack Directly

It can be useful to shortcut things, ie remove Loki for the equation and post to Slack directly.

## `loki2slack post`

To do this use `loki2slack post -c <config_file> --sample-file <example>`.
The example file must have a least two line.
The first is the Loki link and the second the log line.
These can be see when running tail with debug (`loki2slack tail --debug ...`).

**TODO**
- include an example.
- make this easier.

## Curling Slack

The following is the equalent sequence of call `loki2slack` makes.

On startup the bot joins the channel.
This is required as there is no Slack scope for uploading a file to a channel without being being a member.

```
curl \
-F channel=$SLACK_CHANNEL_ID \
-H "Authorization: Bearer $SLACK_TOKEN" \
https://slack.com/api/conversations.join
```

For each log entry the line is uploaded.

```
curl -v \
-F file=@log_entry.txt \
-F "filetype=json" \
-F channels=$SLACK_CHANNEL_ID \
-H "Authorization: Bearer $SLACK_TOKEN" \
https://slack.com/api/files.upload
```

The message with the upload is then updated with the link and labels.

```
curl -v \
-H "Content-type: application/json" \
-H "Authorization: Bearer $SLACK_TOKEN" \
-d '{
  "channel": "$SLACK_CHANNEL_ID",
  "ts": "1638773134.003000", <---   From the upload response
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "<http://localhost:3000/explore?left=%5B%221638440707959%22%2C%221638440707959%22%2C%22Loki%22%2C%7B%22expr%22%3A%22%7Benv%3D%5C%22pvt1%5C%22%7D%22%7D%5D|Grafana Link>"
      }
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "Env: dev \n Release: 0.1.2"
      }
    }
  ]
}' \
https://slack.com/api/chat.update
```