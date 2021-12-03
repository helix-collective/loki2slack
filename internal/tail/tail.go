package tail

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"onederful/loki_fwder/internal/types"

	"github.com/golang/glog"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/slack-go/slack"
	"google.golang.org/grpc"
)

type urldecoderOpts struct {
	rt     *types.Root
	Encode bool
}

func NewDecoder(rt *types.Root) interface{} {
	in := urldecoderOpts{
		rt: rt,
	}
	return &in
}

func (in *urldecoderOpts) Run() error {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	// str, err := strconv.Unquote(string(b))
	// if err != nil {
	// 	glog.Errorf("Unquote %v", err)
	// }

	if in.Encode {
		str := url.QueryEscape(string(b))
		fmt.Printf("%s\n", str)
		return nil
	}
	str, err := url.QueryUnescape(string(b))
	// str, err = url.QueryUnescape(str)
	if err != nil {
		glog.Errorf("QueryUnescape %v", err)
		return err
	}
	fmt.Printf("%s\n", str)
	return err
}

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
		GrafanaUrl:     "http://localhost:3001",
		Query:          `{env="devel"}`,
	}
	return &in
}

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
	config(in.Cfg, in.DumpConfig, in)

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

func config(filename string, dump bool, in interface{}) {
	if filename != "" {
		fd, err := os.Open(filename)
		// config is in its own func
		// this defer fire correctly
		//
		// won't fire if dump is used as os.Exit terminates program
		defer func() {
			err := fd.Close()
			glog.Infof("close file %v", err)
		}()
		if err != nil {
			log.Fatalf("error opening file %s %v", filename, err)
		}
		dec := json.NewDecoder(fd)
		err = dec.Decode(in)
		if err != nil {
			log.Fatalf("json error %v", err)
		}
	}
	if dump {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		err := enc.Encode(in)
		if err != nil {
			log.Fatalf("json encoding error %v", err)
		}
		os.Exit(0)
	}
}

func (in *tailOpts) Run() error {
	config(in.Cfg, in.DumpConfig, in)
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
