module github.com/reubenmiller/go-c8y

replace github.com/reubenmiller/go-c8y/pkg/c8y => ./pkg/c8y

replace github.com/reubenmiller/go-c8y/pkg/microservice => ./pkg/microservice

replace github.com/reubenmiller/go-c8y/test/c8y_test => ./test/c8y_test

replace github.com/reubenmiller/go-c8y/test/c8y_microservice => ./test/c8y_microservice

require (
	github.com/Djarvur/go-err113 v0.1.0 // indirect
	github.com/araddon/dateparse v0.0.0-20190329160016-74dc0e29b01f
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/golangci/golangci-lint v1.31.0 // indirect
	github.com/golangci/misspell v0.3.5 // indirect
	github.com/golangci/revgrep v0.0.0-20180812185044-276a5c0a1039 // indirect
	github.com/google/go-jsonnet v0.16.0
	github.com/google/go-querystring v1.0.0
	github.com/gorilla/websocket v1.4.2
	github.com/gostaticanalysis/analysisutil v0.2.1 // indirect
	github.com/jirfag/go-printf-func-name v0.0.0-20200119135958-7558a9eaa5af // indirect
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.8 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/matoous/godox v0.0.0-20200801072554-4fb83dc2941e // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/obeattie/ohmyglob v0.0.0-20150811221449-290764208a0d
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v0.9.3
	github.com/quasilyte/regex/syntax v0.0.0-20200805063351-8f842688393c // indirect
	github.com/simplereach/timeutils v1.2.0 // indirect
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/spf13/afero v1.4.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/tdakkota/asciicheck v0.0.0-20200416200610-e657995f937b // indirect
	github.com/tetafro/godot v0.4.9 // indirect
	github.com/tidwall/gjson v1.2.1
	github.com/tidwall/match v1.0.1 // indirect
	github.com/tidwall/pretty v0.0.0-20190325153808-1166b9ac2b65 // indirect
	github.com/timakin/bodyclose v0.0.0-20200424151742-cb6215831a94 // indirect
	github.com/valyala/fasttemplate v1.0.1 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/sys v0.0.0-20200923182605-d9f96fdee20d // indirect
	golang.org/x/tools v0.0.0-20200923182640-463111b69878 // indirect
	gopkg.in/ini.v1 v1.61.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	moul.io/http2curl v1.0.0
	mvdan.cc/gofumpt v0.0.0-20200802201014-ab5a8192947d // indirect
	mvdan.cc/unparam v0.0.0-20200501210554-b37ab49443f7 // indirect
)

go 1.13
