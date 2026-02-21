module github.com/reubenmiller/example

go 1.25

require (
	github.com/alecthomas/kong v1.12.1
	github.com/eclipse/paho.golang v0.22.0
	github.com/reubenmiller/go-c8y v0.0.0-00010101000000-000000000000
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/net v0.43.0 // indirect
)

replace github.com/reubenmiller/go-c8y => ../../
