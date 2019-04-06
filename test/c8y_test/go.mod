module github.com/reubenmiller/go-c8y/test/c8y_test

require (
	github.com/pkg/errors v0.8.1
	github.com/reubenmiller/go-c8y v0.4.0
	github.com/reubenmiller/go-c8y/pkg/c8y v0.0.0-20190401191817-d4dface78e96
)

replace github.com/reubenmiller/go-c8y => ../..

replace github.com/reubenmiller/go-c8y/pkg/c8y => ../../pkg/c8y
