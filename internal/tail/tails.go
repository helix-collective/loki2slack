package tail

import (
	"context"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/helix-collective/loki2slack/internal/posttmplt"
	"github.com/helix-collective/loki2slack/internal/types"
)

type tailsOpts struct {
	rt         *types.Root
	Cfg        string `help:"Config file in json format" json:"-"`
	DumpConfig bool   `help:"Dump the config to stdout and exits" json:"-"`
	Debug      bool   `opts:"-"`

	Addr           string     `opts:"-"`
	LokiDataSource string     `opts:"-"`
	GrafanaUrl     string     `opts:"-"`
	Tail           []tailSpec `opts:"-"`
}

type tailSpec struct {
	Query          string
	Templates      []string
	TemplateFile   string `help:"Filename of template. Expected are templates name 'message' (required), 'json_attachment' & 'txt_attachment'."`
	SlackToken     string `opts:"env" help:"make sure scope chat:write is added (So far only working with user token)"`
	SlackChannelId string `opts:"env" help:"copy channel from the bottom on 'open channel details' dialogue"`
}

func NewTails(rt *types.Root) interface{} {
	in := tailsOpts{
		rt:             rt,
		Addr:           "localhost:9096",
		LokiDataSource: "Loki",
		GrafanaUrl:     "http://localhost:3000",
		Tail: []tailSpec{
			{
				Query:     `{env="dev1"}`,
				Templates: []string{},
			},
			{
				Query:     `{env="dev2"}`,
				Templates: []string{},
			},
		},
	}
	return &in
}

const TailsUsage = `Tails loki multiple times, one for each tail entry in the config.
See 'tail --help' and 'tails --dump-config' for more information.`

func (in *tailsOpts) Run() error {
	types.Config(in.Cfg, in.DumpConfig, in)
	if in.Cfg == "" {
		glog.Fatalf("--cfg required")
	}
	for i0, t0 := range in.Tail {
		go func(t1 tailSpec, i1 int) {
			t2 := tailOpts{
				rt:             in.rt,
				Debug:          in.Debug,
				Addr:           in.Addr,
				LokiDataSource: in.LokiDataSource,
				GrafanaUrl:     in.GrafanaUrl,
				Query:          t1.Query,
				TemplateFile:   t1.TemplateFile,
				SlackToken:     t1.SlackToken,
				SlackChannelId: t1.SlackChannelId,
			}
			tmplMap := make(map[string]string)
			for _, tmplStr := range t1.Templates {
				idx := strings.Index(tmplStr, ":")
				if idx == -1 {
					glog.Fatalf("expected ':' in template '%s'", tmplStr)
				}
				tmplMap[tmplStr[:idx]] = tmplStr[idx+1:]
				if in.Debug {
					glog.Infof("template '%s' '%s'", tmplStr[:idx], tmplStr[idx+1:])
				}
			}
			tmpl, err := posttmplt.ParseTemplate(t1.TemplateFile, tmplMap)
			if err != nil {
				glog.Errorf("error parsing template tail %d '%s' %v", i1, t2.TemplateFile, err)
				return
			}
			for {
				t2.tailLoki(tmpl)
				glog.Infof("waiting and reconnecting, tail %d", i1)
				<-time.After(time.Second * 5)
			}
		}(t0, i0)
	}
	glog.Info("tails waiting")
	<-context.Background().Done()
	return nil
}
