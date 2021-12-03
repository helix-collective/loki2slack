package tail

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/slack-go/slack"
)

// from https://github.com/slack-go/slack/blob/master/examples/messages/messages.go
func postMsg(env string, lokiLink string, lokiLine string, Debug bool, SlackChannelId string, SlackToken string) error {
	if len(lokiLine) > (1000 - 6) {
		lokiLine = lokiLine[:(1000 - 6)]
	}
	lokiLine = strings.ReplaceAll(lokiLine, `\n`, "\n")
	lokiLine = strings.ReplaceAll(lokiLine, `\t`, "\t")
	lokiLine = strings.ReplaceAll(lokiLine, `\"`, "\"")
	fmt.Printf("link: %d\n%s\n\nline: %d\n%s\n", len(lokiLink), lokiLink, len(lokiLine), lokiLine)

	headerText := slack.NewTextBlockObject(
		"mrkdwn",
		fmt.Sprintf("Environment %s", env),
		false,
		false,
	)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)

	link := slack.NewTextBlockObject(
		"mrkdwn",
		fmt.Sprintf("%s\n", lokiLink),
		false,
		true,
	)
	body := slack.NewTextBlockObject(
		"mrkdwn",
		fmt.Sprintf("```%s```", lokiLine),
		false,
		true,
	)

	// fieldSlice := make([]*slack.TextBlockObject, 0)
	// fieldSlice = append(fieldSlice, link)
	// fieldSlice = append(fieldSlice, body)
	// fieldsSection := slack.NewSectionBlock(nil, fieldSlice, nil)

	msg := slack.NewBlockMessage(
		headerSection,
		slack.NewSectionBlock(link, nil, nil),
		slack.NewSectionBlock(body, nil, nil),
		// fieldsSection,
	)

	if Debug {
		b, err := json.MarshalIndent(msg, "", "    ")
		if err != nil {
			glog.Warning(err)
		} else {
			glog.Info(string(b))
		}
	}

	api := slack.New(SlackToken)
	// attachment := slack.Attachment{
	// 	Pretext: "Entry Line",
	// 	Text:    "Entry Line",
	// 	// Uncomment the following part to send a field too
	// 	Fields: []slack.AttachmentField{
	// 		slack.AttachmentField{
	// 			Title: "a",
	// 			Value: "no",
	// 		},
	// 	},
	// }
	channelID, timestamp, err := api.PostMessage(
		SlackChannelId,
		slack.MsgOptionBlocks(msg.Blocks.BlockSet...),
		// slack.MsgOptionAttachments(attachment),
		// Add this if you want that the bot would post message as a user,
		// otherwise it will send response using the default slackbot
		slack.MsgOptionAsUser(false),
	)
	if err != nil {
		glog.Warningf("%s", err)
		return err
	}
	glog.Infof("Message successfully sent to channel %s at %s\n", channelID, timestamp)
	return nil
}
