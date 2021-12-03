package tail

import (
	"bufio"
	"os"

	"github.com/golang/glog"

	"github.com/helix-collective/loki2slack/internal/types"
)

type postOpts struct {
	rt         *types.Root
	Cfg        string `help:"Config file in json format (NOTE file entries take precedence over command-line flags & env)" json:"-"`
	DumpConfig bool   `help:"Dump the config to stdout and exits" json:"-"`
	Debug      bool

	SlackToken     string `opts:"env" help:"make sure scope chat:write is added (So far only working with user token)"`
	SlackChannelId string `opts:"env" help:"copy channel from the bottom on 'open channel details' dialogue"`

	SampleFile string
}

func NewPost(rt *types.Root) interface{} {
	in := postOpts{
		rt: rt,
	}
	return &in
}

func (in *postOpts) Run() error {
	types.Config(in.Cfg, in.DumpConfig, in)

	fd, err := os.Open(in.SampleFile)
	if err != nil {
		glog.Fatalf("error opening file %s %v", in.SampleFile, err)
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	scanner.Scan()
	lokiLink := scanner.Text()
	scanner.Scan()
	lokiLine := scanner.Text()
	return postMsg(
		`"testing"`,
		lokiLink,
		lokiLine,
		in.Debug,
		in.SlackChannelId,
		in.SlackToken,
	)
}
