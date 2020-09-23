module github.com/reubenmiller/go-c8y

replace github.com/reubenmiller/go-c8y/pkg/microservice => ./pkg/microservice

replace github.com/reubenmiller/go-c8y/test/c8y_test => ./test/c8y_test

replace github.com/reubenmiller/go-c8y/test/c8y_microservice => ./test/c8y_microservice

require (
	github.com/araddon/dateparse v0.0.0-20190329160016-74dc0e29b01f
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/fatih/color v1.9.0
	github.com/google/go-querystring v1.0.0
	github.com/gorilla/websocket v1.4.0
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/jeremywohl/flatten v1.0.1 // indirect
	github.com/karrick/tparse/v2 v2.8.2
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.8 // indirect
	github.com/manifoldco/promptui v0.7.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/obeattie/ohmyglob v0.0.0-20150811221449-290764208a0d
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3
	github.com/simplereach/timeutils v1.2.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.4.0
	github.com/thedevsaddam/gojsonq v2.3.0+incompatible
	github.com/tidwall/gjson v1.2.1
	github.com/tidwall/match v1.0.1 // indirect
	github.com/tidwall/pretty v0.0.0-20190325153808-1166b9ac2b65
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8 // indirect
	github.com/valyala/fasttemplate v1.0.1 // indirect
	go.uber.org/zap v1.10.0
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	moul.io/http2curl v1.0.0
)

go 1.13
