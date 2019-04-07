module github.com/reubenmiller/go-c8y

require (
	github.com/reubenmiller/go-c8y/pkg/c8y v0.0.0-20190401191817-d4dface78e96
	github.com/reubenmiller/go-c8y/pkg/microservice v0.0.0-20190401160529-913a4f3caafc
	github.com/spf13/viper v1.3.2
)

replace github.com/reubenmiller/go-c8y/pkg/microservice => ./pkg/microservice

replace github.com/reubenmiller/go-c8y/test/c8y_test => ./test/c8y_test

replace github.com/reubenmiller/go-c8y/test/c8y_microservice => ./test/c8y_microservice
