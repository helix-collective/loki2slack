package tail

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/helix-collective/loki2slack/internal/posttmplt"
	"github.com/helix-collective/loki2slack/internal/types"
	"github.com/slack-go/slack"

	"github.com/golang/glog"
	"github.com/grafana/loki/pkg/logproto"
	"google.golang.org/grpc"
)

type tailOpts struct {
	rt         *types.Root
	Cfg        string `help:"Config file in json format (NOTE file entries take precedence over command-line flags & env)" json:"-"`
	DumpConfig bool   `help:"Dump the config to stdout and exits" json:"-"`
	Debug      bool

	Addr           string
	LokiDataSource string
	GrafanaUrl     string
	Query          string

	TemplateFile string `help:"Filename of template. Expected are templates name 'message' (required), 'json_attachment' & 'txt_attachment'."`

	SlackToken     string `opts:"env" help:"make sure scope chat:write is added (So far only working with user token)"`
	SlackChannelId string `opts:"env" help:"copy channel from the bottom on 'open channel details' dialogue"`
}

func (in *tailOpts) GetDebug() bool            { return in.Debug }
func (in *tailOpts) GetQuery() string          { return in.Query }
func (in *tailOpts) GetSlackChannelId() string { return in.SlackChannelId }
func (in *tailOpts) GetSlackToken() string     { return in.SlackToken }
func (in *tailOpts) GetGrafanaUrl() string     { return in.GrafanaUrl }
func (in *tailOpts) GetLokiDataSource() string { return in.LokiDataSource }

// New constructor for init
func New(rt *types.Root) interface{} {
	in := tailOpts{
		rt:             rt,
		Addr:           "localhost:9096",
		LokiDataSource: "Loki",
		GrafanaUrl:     "http://localhost:3000",
		Query:          `{env="dev"}`,
	}
	return &in
}

func (in *tailOpts) Run() error {
	types.Config(in.Cfg, in.DumpConfig, in)
	tmpl, err := posttmplt.ParseTemplate(in.TemplateFile)
	if err != nil {
		glog.Fatalf("error parsing template '%s' %v", in.TemplateFile, err)
	}
	for {
		in.tailLoki(tmpl)
		glog.Info("waiting and reconnecting")
		<-time.After(time.Second * 5)
	}
}

func (in *tailOpts) tailLoki(tmpl *template.Template) error {
	go func() {
		api := slack.New(in.SlackToken)
		_, _, _, err := api.JoinConversation(in.SlackChannelId)
		if err == nil {
			glog.Info("joinChannel ok")
		} else {
			glog.Warningf("joinChannel error %v", err)
		}
	}()
	// connection to Loki
	conn, err := grpc.Dial(in.Addr, grpc.WithInsecure())
	if err != nil {
		glog.Warningf("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	qc := logproto.NewQuerierClient(conn)
	ctx := context.Background()
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()
	tr, err := qc.Tail(ctx, &logproto.TailRequest{
		Query: in.Query,
	})
	if err != nil {
		glog.Warningf("call tail: %v", err)
		return err
	}
	glog.Infof("connected tailing loki with query %s", in.Query)
	for {
		tresp, err := tr.Recv()
		if err != nil {
			glog.Warningf("call tail: %v", err)
			return err
		}
		for _, entry := range tresp.DroppedStreams {
			glog.Warningf("dropped %v\n", entry)
		}
		// fmt.Printf(">> %s\n", tresp.Stream.Labels)
		plabels := tresp.Stream.Labels
		// remove leading and tralling '{' '}'
		plabels = plabels[1 : len(plabels)-1]

		pl := strings.Split(plabels, ",")
		labelData := make(map[string]interface{})

		for _, p := range pl {
			idx := strings.Index(p, "=")
			val := p[idx+2:]
			if val[len(val)-1] == '"' {
				val = val[:len(val)-1]
			}
			if len(val) < 80 {
				labelData[p[:idx]] = val
			}
		}

		for _, entry := range tresp.Stream.Entries {
			msg, att, err := posttmplt.ProcessTemplate(
				in,
				tmpl,
				labelData,
				[]byte(entry.Line),
				entry.Timestamp.UnixMilli(),
			)
			if err != nil {
				glog.Warningf("ProcessTemplate error %v", err)
				continue
			}
			msgStr := msg.String()
			if in.Debug {
				fmt.Printf("``` message\n%s\n```\n", msgStr)
			}
			if att != nil {
				attStr := att.String()
				if in.Debug {
					fmt.Printf("``` attachment\n%s\n```\n", attStr)
				}
				posttmplt.Post(in, msgStr, &attStr)
			} else {
				posttmplt.Post(in, msgStr, nil)
			}
		}
	}
	// http://localhost:3001/explore?left=["1638359982286","1638360882286","Loki",{"expr":"{env=\"devel\"}"}]&orgId=1
}
