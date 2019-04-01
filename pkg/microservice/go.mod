module github.com/reubenmiller/go-c8y/pkg/microservice

require (
	// github.com/reubenmiller/go-c8y/pkg/c8y v0.3.0
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/araddon/dateparse v0.0.0-20181123171228-21df004e09ca
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.8 // indirect
	github.com/mattn/go-colorable v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/reubenmiller/go-c8y/pkg/c8y v0.0.0-20190401171348-83ae0b671f00
	github.com/simplereach/timeutils v1.2.0 // indirect
	github.com/spf13/viper v1.2.1
	github.com/tidwall/gjson v1.1.3
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v0.0.0-20170224212429-dcecefd839c4 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20190211182817-74369b46fc67 // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
)

replace github.com/reubenmiller/go-c8y/pkg/c8y => ../../pkg/c8y
