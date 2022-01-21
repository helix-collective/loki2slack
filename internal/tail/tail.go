package tail

import (
	"context"
	"time"

	"github.com/helix-collective/loki2slack/internal/posttmplt"
	"github.com/helix-collective/loki2slack/internal/slackclient"
	"github.com/helix-collective/loki2slack/internal/types"

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

	TemplateMsgFile        string `help:"Filename of template used for slack message body. Required"`
	TemplateAttachmentFile string `help:"Filename of template used for slack attachement content. Optional"`

	SlackToken     string `opts:"env" help:"make sure scope chat:write is added (So far only working with user token)"`
	SlackChannelId string `opts:"env" help:"copy channel from the bottom on 'open channel details' dialogue"`
}

func (in *tailOpts) GetDebug() bool                    { return in.Debug }
func (in *tailOpts) GetSlackChannelId() string         { return in.SlackChannelId }
func (in *tailOpts) GetSlackToken() string             { return in.SlackToken }
func (in *tailOpts) GetGrafanaUrl() string             { return in.GrafanaUrl }
func (in *tailOpts) GetLokiDataSource() string         { return in.LokiDataSource }
func (in *tailOpts) GetTemplateMsgFile() string        { return in.TemplateMsgFile }
func (in *tailOpts) GetTemplateAttachmentFile() string { return in.TemplateAttachmentFile }

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
	for {
		in.tailLoki()
		glog.Info("waiting and reconnecting")
		<-time.After(time.Second * 5)
	}
}

func (in *tailOpts) tailLoki() error {
	go func() {
		err := slackclient.JoinChannel(in.SlackChannelId, in.SlackToken)
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
		for _, entry := range tresp.Stream.Entries {
			msg, att, err := posttmplt.ProcessTemplate(
				in,
				[]byte(plabels),
				[]byte(entry.Line),
				entry.Timestamp.UnixMilli(),
			)
			if err != nil {
				glog.Warningf("ProcessTemplate error %v", err)
				continue
			}
			posttmplt.Post(in, msg, att)
		}
	}
	// http://localhost:3001/explore?left=["1638359982286","1638360882286","Loki",{"expr":"{env=\"devel\"}"}]&orgId=1
}
