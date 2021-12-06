package tail

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/slack-go/slack"
)

func joinChannel(SlackChannelId string, SlackToken string) error {
	api := slack.New(SlackToken)
	_, _, _, err := api.JoinConversation(SlackChannelId)
	return err
}

func uploadFile(lokiLine string, Debug bool, SlackChannelId string, SlackToken string) (string, error) {
	api := slack.New(SlackToken)
	lokiLine = strings.ReplaceAll(lokiLine, `\n`, "\n")
	lokiLine = strings.ReplaceAll(lokiLine, `\t`, "\t")
	lokiLine = strings.ReplaceAll(lokiLine, `\"`, "\"")
	file, err := api.UploadFile(slack.FileUploadParameters{
		Content:  lokiLine,
		Channels: []string{SlackChannelId},
		Filetype: "json",
		Filename: "Log Entry",
	})
	if err != nil {
		glog.Warningf("Error uploading file %v", err)
		return "", err
	}
	ts := file.Shares.Public[SlackChannelId][0].Ts
	glog.Infof("ts: %v", ts)
	return ts, nil
}

func updateMsg(SlackChannelId string, SlackToken string, ts string, link string, labels []string) {
	api := slack.New(SlackToken)
	msg := slack.NewTextBlockObject(
		"mrkdwn",
		link+"\n"+strings.Join(labels, "\n"),
		false,
		true,
	)
	api.UpdateMessage(SlackChannelId, ts,
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(msg, nil, nil),
		),
	)
}

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
