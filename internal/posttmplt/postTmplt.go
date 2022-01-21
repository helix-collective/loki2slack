package posttmplt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"github.com/helix-collective/loki2slack/internal/slackclient"
	"github.com/helix-collective/loki2slack/internal/types"
)

type postTmplOpts struct {
	rt             *types.Root
	Cfg            string `help:"Config file in json format (NOTE file entries take precedence over command-line flags & env)" json:"-"`
	DumpConfig     bool   `help:"Dump the config to stdout and exits" json:"-"`
	Debug          bool
	DryRun         bool
	LokiDataSource string
	GrafanaUrl     string

	SlackToken     string `opts:"env" help:"make sure scope chat:write is added (So far only working with user token)"`
	SlackChannelId string `opts:"env" help:"copy channel from the bottom on 'open channel details' dialogue"`

	TemplateMsgFile        string `help:"Filename of template used for slack message body. Required"`
	TemplateAttachmentFile string `help:"Filename of template used for slack attachement content. Optional"`

	SampleLabelsFile string `help:"Filename of sample labels. Format (name=\"value\"\\n)* "`
	SampleLineFile   string `help:"Filename of sample line. Format = json."`
}

func (in *postTmplOpts) GetDebug() bool                    { return in.Debug }
func (in *postTmplOpts) GetSlackChannelId() string         { return in.SlackChannelId }
func (in *postTmplOpts) GetSlackToken() string             { return in.SlackToken }
func (in *postTmplOpts) GetGrafanaUrl() string             { return in.GrafanaUrl }
func (in *postTmplOpts) GetLokiDataSource() string         { return in.LokiDataSource }
func (in *postTmplOpts) GetTemplateMsgFile() string        { return in.TemplateMsgFile }
func (in *postTmplOpts) GetTemplateAttachmentFile() string { return in.TemplateAttachmentFile }

const PostTemplateUsage = `Data available to the template engine.
struct {
	GrafanaUrl     string
	EntryTimestamp int64
	LokiDataSource string
	Labels         map[string]interface{}
	Line           interface{}
}
Labels are the log labels from Loki.
If the Line is json formatted then its type can be assumed as map[string]interface{}.

Example template
` + "```" + `
  {{.Labels.env}}
  {{.Line.body}}
  {{$left := printf ` + "`" + `["%d","%d","%s",{"expr":"{env=\"%s\"}"}]` + "`" + `
         .EntryTimestamp .EntryTimestamp .LokiDataSource .Labels.env
     | urlquery
  }}
  {{printf "%[1]s/explore?left=%[2]s" .GrafanaUrl $left}}
  {{ range $key, $value := .Line }}
  {{ $key }}: {{ $value }}
  {{- end }}
` + "```"

func NewPostTemplate(rt *types.Root) interface{} {
	in := postTmplOpts{
		rt:             rt,
		LokiDataSource: "Loki",
		GrafanaUrl:     "http://localhost:3000",
	}
	return &in
}

func (in *postTmplOpts) Run() error {
	types.Config(in.Cfg, in.DumpConfig, in)

	labelsTxt, err := ioutil.ReadFile(in.SampleLabelsFile)
	if err != nil {
		glog.Fatalf("error opening file %s %v", in.SampleLabelsFile, err)
	}

	lineTxt, err := ioutil.ReadFile(in.SampleLineFile)
	if err != nil {
		glog.Fatalf("error opening file %s %v", in.SampleLineFile, err)
	}

	now := time.Now().UnixMilli()
	msg, att, err := ProcessTemplate(in, labelsTxt, lineTxt, now)
	if err != nil {
		return err
	}
	if in.DryRun {
		print("message\n```\n" + msg.String() + "\n```\n")
		if att != nil {
			print("attachment\n```\n" + att.String() + "\n```\n")
		}
		return nil
	}
	return Post(in, msg, att)
}

type PostTempParams interface {
	GetDebug() bool
	GetSlackChannelId() string
	GetSlackToken() string
	GetGrafanaUrl() string
	GetLokiDataSource() string
	GetTemplateMsgFile() string
	GetTemplateAttachmentFile() string
}

func Post(in PostTempParams, msg *bytes.Buffer, att *bytes.Buffer) error {
	err := slackclient.JoinChannel(in.GetSlackChannelId(), in.GetSlackToken())
	if err == nil {
		glog.Info("joinChannel ok")
	} else {
		glog.Warningf("joinChannel error %v", err)
		return err
	}

	msgBlk := slack.NewTextBlockObject(
		"mrkdwn",
		msg.String(),
		false,
		true,
	)
	api := slack.New(in.GetSlackToken())
	if att != nil {
		file, err := api.UploadFile(slack.FileUploadParameters{
			Content:  att.String(),
			Channels: []string{in.GetSlackChannelId()},
			Filetype: "json",
			Filename: "Log Entry",
		})
		if err != nil {
			glog.Warningf("Error uploading file %v", err)
			return err
		}
		ts := file.Shares.Public[in.GetSlackChannelId()][0].Ts
		_, _, _, err = api.UpdateMessage(in.GetSlackChannelId(), ts,
			slack.MsgOptionBlocks(
				slack.NewSectionBlock(msgBlk, nil, nil),
			),
		)
		return err
	}
	// no attachement only a message
	_, _, err = api.PostMessage(
		in.GetSlackChannelId(),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(msgBlk, nil, nil),
		),
	)
	return err
}

func ProcessTemplate(
	in PostTempParams,
	labelsTxt []byte,
	lineTxt []byte,
	entryTimestamp int64,
) (*bytes.Buffer, *bytes.Buffer, error) {
	labelData := make(map[string]interface{})
	scanner := bufio.NewScanner(strings.NewReader(string(labelsTxt)))
	for scanner.Scan() {
		label := scanner.Text()
		idx := strings.Index(label, "=")
		labelData[label[:idx]] = label[idx+2 : len(label)-1]
	}

	var lineData interface{}
	lineMapData := make(map[string]interface{})
	err := json.Unmarshal(lineTxt, &lineMapData)
	if err != nil {
		glog.Warningf("json 'line' expected %v", err)
		if in.GetDebug() {
			glog.Infof("line '%s'", string(lineTxt))
		}
		lineData = string(lineTxt)
	} else {
		lineData = lineMapData
	}

	data := struct {
		GrafanaUrl     string
		EntryTimestamp int64
		LokiDataSource string
		Labels         map[string]interface{}
		Line           interface{}
	}{
		GrafanaUrl:     in.GetGrafanaUrl(),
		EntryTimestamp: entryTimestamp,
		LokiDataSource: in.GetLokiDataSource(),
		Labels:         labelData,
		Line:           lineData,
	}
	msgBuf := &bytes.Buffer{}
	{
		tmpl, err := template.ParseFiles(in.GetTemplateMsgFile())
		if err != nil {
			glog.Warningf("msg template error %v %s", err, in.GetTemplateMsgFile())
			return nil, nil, err
		}
		err = tmpl.Execute(msgBuf, data)
		if err != nil {
			glog.Warningf("error exec msg template %v", err)
			return nil, nil, err
		}
		if in.GetTemplateAttachmentFile() == "" {
			return msgBuf, nil, nil
		}
	}
	at_tmpl, err := template.ParseFiles(in.GetTemplateAttachmentFile())
	if err != nil {
		glog.Warningf("attachment template error %v %s", err, in.GetTemplateAttachmentFile())
		return nil, nil, err
	}
	attBuf := &bytes.Buffer{}
	err = at_tmpl.Execute(attBuf, data)
	if err != nil {
		glog.Warningf("error exec attachment template %v", err)
		return nil, nil, err
	}
	return msgBuf, attBuf, nil
}
