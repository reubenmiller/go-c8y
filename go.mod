module github.com/reubenmiller/go-c8y

require (
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.2
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/go-jsonnet v0.18.0
	github.com/google/go-querystring v1.1.0
	github.com/gorilla/websocket v1.4.2
	github.com/kr/text v0.2.0 // indirect
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.1 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/obeattie/ohmyglob v0.0.0-20150811221449-290764208a0d
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.0
	github.com/spf13/afero v1.8.0 // indirect
	github.com/spf13/viper v1.10.1
	github.com/tidwall/gjson v1.13.0
	github.com/vbauerster/mpb/v6 v6.0.4
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.20.0
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce // indirect
	golang.org/x/net v0.0.0-20220121210141-e204ce36a2ba
	golang.org/x/tools v0.1.8 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.66.3 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace github.com/reubenmiller/go-c8y/pkg/c8y => ./pkg/c8y

replace github.com/reubenmiller/go-c8y/pkg/microservice => ./pkg/microservice

replace github.com/reubenmiller/go-c8y/test/c8y_test => ./test/c8y_test

replace github.com/reubenmiller/go-c8y/test/c8y_microservice => ./test/c8y_microservice

go 1.13
