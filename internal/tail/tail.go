package tail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

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

	SlackToken     string `opts:"env" help:"make sure scope chat:write is added (So far only working with user token)"`
	SlackChannelId string `opts:"env" help:"copy channel from the bottom on 'open channel details' dialogue"`
}

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
		plabels = plabels[1 : len(plabels)-1]
		pl := strings.Split(plabels, ",")
		env := ""
		for _, p := range pl {
			p = strings.Trim(p, " ")
			if strings.HasPrefix(p, "env=\"") {
				env = p
				break
			}
		}
		if env == "" {
			glog.Warningf("didn't find env in %s", pl)
			continue
		}
		envA := strings.Split(env, "=")
		env = strings.ReplaceAll(env, `"`, `\"`)
		for _, entry := range tresp.Stream.Entries {
			unixT := entry.Timestamp.UnixMilli()
			left := fmt.Sprintf(`["%[2]d","%[2]d","%[3]s",{"expr":"{%[4]s}"}]`, in.GrafanaUrl, unixT, in.LokiDataSource, env)
			lokiLink := fmt.Sprintf("%[1]s/explore?left=%[2]s", in.GrafanaUrl, url.QueryEscape(left))
			line := entry.Line
			lmap := make(map[string]interface{})
			err = json.Unmarshal([]byte(line), &lmap)
			if err != nil {
				glog.Infof("entry.Line is not json %v", err)
			} else {
				ba, err := json.MarshalIndent(lmap, "", "  ")
				if err != nil {
					glog.Warningf("can't MarshalIndent %v", err)
				} else {
					line = string(ba)
				}
			}
			// replacer := strings.NewReplacer(`\\n\\t`, "\n\t", `\n\t`, "\n\t", `\n`, "\n")
			// line = replacer.Replace(line)
			postMsg(
				envA[1],
				fmt.Sprintf("<%s|Grafana Link>", lokiLink),
				line,
				in.Debug,
				in.SlackChannelId,
				in.SlackToken,
			)
		}
	}
	// http://localhost:3001/explore?left=["1638359982286","1638360882286","Loki",{"expr":"{env=\"devel\"}"}]&orgId=1
}
