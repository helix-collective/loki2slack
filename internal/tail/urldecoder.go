package tail

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/helix-collective/loki2slack/internal/types"

	"github.com/golang/glog"
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
