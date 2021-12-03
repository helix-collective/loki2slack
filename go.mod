module github.com/helix-collective/loki2slack

go 1.17

replace github.com/jpillora/opts => github.com/millergarym/opts v1.1.10

// replace github.com/grafana/loki => ./

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/grafana/loki v1.6.1
	github.com/jpillora/opts v1.2.0
	github.com/slack-go/slack v0.10.0
	google.golang.org/grpc v1.42.0
)

require (
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/posener/complete v1.2.2-0.20190308074557-af07aa5181b3 // indirect
	github.com/prometheus/prometheus v1.8.2-0.20200727090838-6f296594a852 // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/sys v0.0.0-20200724161237-0e2f3a69832c // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200724131911-43cab4749ae7 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)
