package posttmplt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"github.com/helix-collective/loki2slack/internal/types"
)

type PostTempParams interface {
	GetDebug() bool
	GetQuery() string
	GetSlackChannelId() string
	GetSlackToken() string
	GetGrafanaUrl() string
	GetLokiDataSource() string
}

type postTmplOpts struct {
	rt             *types.Root
	Cfg            string `help:"Config file in json format (NOTE file entries take precedence over command-line flags & env)" json:"-"`
	DumpConfig     bool   `help:"Dump the config to stdout and exits" json:"-"`
	Debug          bool   `json:",omitempty"`
	DryRun         bool
	LokiDataSource string
	GrafanaUrl     string
	Query          string

	Templates []string `help:"Templates which override the file or defaults templates. In the form '<name>:<template>' eg 'message:{{.Query}}'. To remove attachment templates use --template 'json_attachment:-' --template 'txt_attachment:-'"`

	SlackToken     string `opts:"env" help:"make sure scope chat:write is added (So far only working with user token)"`
	SlackChannelId string `opts:"env" help:"copy channel from the bottom on 'open channel details' dialogue"`

	TemplateFile string `help:"Filename of template. Expected are templates name 'message' (required), 'json_attachment' & 'txt_attachment'."`

	SampleLabelsFile string `help:"Filename of sample labels. Format (name=\"value\"\\n)* "`
	SampleLineFile   string `help:"Filename of sample line. Format = json."`
}

func (in *postTmplOpts) GetDebug() bool            { return in.Debug }
func (in *postTmplOpts) GetQuery() string          { return in.Query }
func (in *postTmplOpts) GetSlackChannelId() string { return in.SlackChannelId }
func (in *postTmplOpts) GetSlackToken() string     { return in.SlackToken }
func (in *postTmplOpts) GetGrafanaUrl() string     { return in.GrafanaUrl }
func (in *postTmplOpts) GetLokiDataSource() string { return in.LokiDataSource }

const PostTemplateUsage = `Data available to the template engine.
struct {
	Query          string
	GrafanaUrl     string
	EntryTimestamp int64
	LokiDataSource string
	Labels         map[string]interface{}
	Line           interface{}
}
Labels are the log labels from Loki.
If the Line is json formatted then its type can be assumed as map[string]interface{}.

Extra function (in addition to https://pkg.go.dev/text/template#hdr-Functions)
escapequotes
    replaces all " with \"

The default template (below) is used if not template file is provided.
If attachment templates are provided a slack message is created with upload file (todo wording)
` + "```" + DefaultTemplate + "```"

func NewPostTemplate(rt *types.Root) interface{} {
	in := postTmplOpts{
		rt:             rt,
		LokiDataSource: "Loki",
		GrafanaUrl:     "http://localhost:3000",
		Query:          `{env="dev"}`,
		Templates:      []string{},
	}
	return &in
}

var template_FuncMap = template.FuncMap{
	"escapequotes": quoteEscaper,
}

func ParseTemplate(
	filename string,
	tmplMap map[string]string,
) (tmpl *template.Template, err error) {
	// tmpl0 := template.New("").Funcs(
	// 	template.FuncMap{
	// 		"escapequotes": quoteEscaper,
	// 	},
	// )
	if filename != "" {
		tmpl, err = template.New("").Funcs(template_FuncMap).ParseFiles(filename)
		if err != nil {
			glog.Warningf("msg template error %v %s", err, filename)
			return nil, err
		}
	} else {
		tmpl, err = template.New("").Funcs(template_FuncMap).Parse(DefaultTemplate)
		if err != nil {
			glog.Fatalf("error in default template %v", err)
		}
	}
	for name, str := range tmplMap {
		glog.Infof("name:template  '%s' : '%s'", name, str)
		if str == "-" {
			glog.Infof("skipping %s", name)
			tp0 := template.New("").Funcs(template_FuncMap)
			for _, tp1 := range tmpl.Templates() {
				if tp1.Name() != name {
					glog.Infof("keeping  '%s'", tp1.Name())
					tp0, _ = tp0.AddParseTree(tp1.Name(), tp1.Tree)
				}
			}
			tmpl = tp0
			continue
		}
		tp, err0 := template.New("").Funcs(template_FuncMap).Parse(str)
		if err0 != nil {
			glog.Errorf("error in msg template %v", err0)
			return nil, err0
		}
		tmpl, _ = tmpl.AddParseTree(name, tp.Tree)
	}
	if tmpl.Lookup("message") == nil {
		return nil, fmt.Errorf("requires 'message' template")
	}
	return tmpl, nil
}

const (
	defauleLabelsTxt = `body="http request failed"
code_version="release-0.86.1"
component="mobileapi/server"
ec2_instance_id="i-1234567890"
env="pvt1"`

	defaultLineTxt = `{
    "threadId": "qtp641030345-1031810318",
    "stacktrace": "java.lang.RuntimeException: Cannot take a connection\n\t note \\n and \\t will get expanded .. 47 more\n",
    "level": "error",
    "logger": "Snuffles",
    "fingerprint": "1234567890",
    "body": "http request failed",
    "url": "http://mobileapi...."
}`
)

func (in *postTmplOpts) Run() error {
	types.Config(in.Cfg, in.DumpConfig, in)

	var labelsTxt []byte
	var err error
	if in.SampleLabelsFile != "" {
		labelsTxt, err = ioutil.ReadFile(in.SampleLabelsFile)
		if err != nil {
			glog.Fatalf("error opening file %s %v", in.SampleLabelsFile, err)
		}
	} else {
		labelsTxt = []byte(defauleLabelsTxt)
	}
	var lineTxt []byte
	if in.SampleLineFile != "" {
		lineTxt, err = ioutil.ReadFile(in.SampleLineFile)
		if err != nil {
			glog.Fatalf("error opening file %s %v", in.SampleLineFile, err)
		}
	} else {
		lineTxt = []byte(defaultLineTxt)
	}

	labelData := make(map[string]interface{})
	scanner := bufio.NewScanner(strings.NewReader(string(labelsTxt)))
	for scanner.Scan() {
		label := scanner.Text()
		idx := strings.Index(label, "=")
		labelData[label[:idx]] = label[idx+2 : len(label)-1]
	}

	tmplMap := make(map[string]string)
	for _, tmplStr := range in.Templates {
		idx := strings.Index(tmplStr, ":")
		if idx == -1 {
			glog.Fatalf("expected ':' in template '%s'", tmplStr)
		}
		tmplMap[tmplStr[:idx]] = tmplStr[idx+1:]
		if in.Debug {
			glog.Infof("template '%s' '%s'", tmplStr[:idx], tmplStr[idx+1:])
		}
	}

	now := time.Now().UnixMilli()
	tmpl, err := ParseTemplate(in.TemplateFile, tmplMap)
	if err != nil {
		glog.Fatalf("error parsing template '%s' %v", in.TemplateFile, err)
	}
	msg, att, err := ProcessTemplate(in, tmpl, labelData, lineTxt, now)
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
	api := slack.New(in.SlackToken)
	_, _, _, err = api.JoinConversation(in.SlackChannelId)
	if err != nil {
		return err
	}
	if att != nil {
		attStr := att.String()
		return Post(in, msg.String(), &attStr)
	}
	return Post(in, msg.String(), nil)
}

func Post(in PostTempParams, msg string, att *string) error {
	msgBlk := slack.NewTextBlockObject(
		"mrkdwn",
		msg,
		false,
		true,
	)
	api := slack.New(in.GetSlackToken())
	if att != nil {
		file, err := api.UploadFile(slack.FileUploadParameters{
			Content:  *att,
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
		if err != nil {
			glog.Warningf("Error updating message %v", err)
			return err
		}
		return nil
	}
	// no attachement only a message
	_, _, err := api.PostMessage(
		in.GetSlackChannelId(),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(msgBlk, nil, nil),
		),
	)
	if err != nil {
		glog.Warningf("Error posting message %v", err)
		return err
	}
	return nil
}

func ProcessTemplate(
	in PostTempParams,
	tmpl *template.Template,
	labelData map[string]interface{},
	lineTxt []byte,
	entryTimestamp int64,
) (*bytes.Buffer, *bytes.Buffer, error) {

	data := struct {
		Query          string
		GrafanaUrl     string
		EntryTimestamp int64
		LokiDataSource string
		Labels         map[string]interface{}
		Line           interface{}
	}{
		Query:          in.GetQuery(),
		GrafanaUrl:     in.GetGrafanaUrl(),
		EntryTimestamp: entryTimestamp,
		LokiDataSource: in.GetLokiDataSource(),
		Labels:         labelData,
	}
	msgBuf := &bytes.Buffer{}
	err := tmpl.Lookup("message").Execute(msgBuf, data)
	if err != nil {
		glog.Warningf("error exec msg template %v", err)
		return nil, nil, err
	}
	var attBuf *bytes.Buffer
	var jsonAtt bool
	if tmpl.Lookup("json_attachment") != nil {
		lineMapData := make(map[string]interface{})
		err := json.Unmarshal(lineTxt, &lineMapData)
		if err != nil {
			glog.Warningf("json 'line' expected %v", err)
			if in.GetDebug() {
				glog.Infof("line '%s'", string(lineTxt))
			}
			jsonAtt = false
		} else {
			data.Line = lineMapData
			jsonAtt = true
			attBuf = &bytes.Buffer{}
			err = tmpl.Lookup("json_attachment").Execute(attBuf, data)
			if err != nil {
				glog.Warningf("error exec json attachment template %v", err)
				return nil, nil, err
			}
		}
	}
	if tmpl.Lookup("txt_attachment") != nil && (tmpl.Lookup("json_attachment") == nil || !jsonAtt) {
		attBuf = &bytes.Buffer{}
		data.Line = string(lineTxt)
		err = tmpl.Lookup("txt_attachment").Execute(attBuf, data)
		if err != nil {
			glog.Warningf("error exec txt attachment template %v", err)
			return nil, nil, err
		}
	}
	return msgBuf, attBuf, nil
}
