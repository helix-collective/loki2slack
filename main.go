package main

import (
	"fmt"

	"github.com/helix-collective/loki2slack/internal/tail"
	"github.com/helix-collective/loki2slack/internal/types"
	"github.com/jpillora/opts"
)

func main() {
	rflg := &types.Root{}
	op := opts.New(rflg).
		Name("loki2slack").
		EmbedGlobalFlagSet().
		Complete().
		AddCommand(opts.New(&versionCmd{}).Name("version")).
		AddCommand(opts.New(tail.New(rflg)).Name("tail")).
		AddCommand(opts.New(tail.NewDecoder(rflg)).Name("urldecode")).
		AddCommand(opts.New(tail.NewPost(rflg)).Name("post")).
		Parse()
	op.RunFatal()
}

// Set by build tool chain by
// go build --ldflags '-X main.Version=xxx -X main.Date=xxx -X main.Commit=xxx'
var (
	Version string = "dev"
	Date    string = "na"
	Commit  string = "na"
)

type versionCmd struct{}

func (r *versionCmd) Run() {
	fmt.Printf("version: %s\ndate: %s\ncommit: %s\n", Version, Date, Commit)
}
