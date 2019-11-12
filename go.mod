module github.com/reubenmiller/go-c8y

replace github.com/reubenmiller/go-c8y/pkg/microservice => ./pkg/microservice

replace github.com/reubenmiller/go-c8y/test/c8y_test => ./test/c8y_test

replace github.com/reubenmiller/go-c8y/test/c8y_microservice => ./test/c8y_microservice

require (
	github.com/araddon/dateparse v0.0.0-20190329160016-74dc0e29b01f
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/fatih/color v1.7.0
	github.com/gohugoio/hugo v0.58.3 // indirect
	github.com/google/go-querystring v1.0.0
	github.com/gorilla/websocket v1.4.0
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/jeremywohl/flatten v0.0.0-20190921043622-d936035e55cf // indirect
	github.com/karrick/tparse v2.4.2+incompatible
	github.com/karrick/tparse/v2 v2.7.1
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.8 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/obeattie/ohmyglob v0.0.0-20150811221449-290764208a0d
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3
	github.com/reubenmiller/promptui v0.3.3-0.20191108115601-2f95ff728c78
	github.com/simplereach/timeutils v1.2.0 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.5.0
	github.com/thedevsaddam/gojsonq v2.3.0+incompatible
	github.com/tidwall/gjson v1.2.1
	github.com/tidwall/match v1.0.1 // indirect
	github.com/tidwall/pretty v1.0.0
	github.com/valyala/fasttemplate v1.0.1 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191029031824-8986dd9e96cf
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	moul.io/http2curl v1.0.0
)

replace github.com/manifoldco/promptui => github.com/reubenmiller/promptui v0.3.3-0.20191108135340-17a79c13fae0

go 1.13
