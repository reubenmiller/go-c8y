module github.com/reubenmiller/go-c8y

require (
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/go-jsonnet v0.21.0
	github.com/google/go-querystring v1.1.0
	github.com/gorilla/websocket v1.5.3
	github.com/labstack/echo/v4 v4.13.4
	github.com/mdp/qrterminal/v3 v3.2.1
	github.com/obeattie/ohmyglob v0.0.0-20150811221449-290764208a0d
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.22.0
	github.com/spf13/viper v1.20.1
	github.com/tidwall/gjson v1.18.0
	github.com/vbauerster/mpb/v8 v8.10.2
	go.mozilla.org/pkcs7 v0.9.0
	go.uber.org/zap v1.27.0
	golang.org/x/net v0.41.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
)

require (
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/qr v0.2.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/reubenmiller/go-c8y/pkg/c8y => ./pkg/c8y

replace github.com/reubenmiller/go-c8y/pkg/microservice => ./pkg/microservice

replace github.com/reubenmiller/go-c8y/test/c8y_test => ./test/c8y_test

replace github.com/reubenmiller/go-c8y/test/c8y_microservice => ./test/c8y_microservice

go 1.24.6
