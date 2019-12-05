# Realtime cli example

## Building

The following steps will create the binary "main" (on Linux/MacOS) or "main.exe" on Windows.

### Windows

**cmd.exe**

```sh
SET GO111MODULE=on
go build main.go
```

**powershell.exe**

```sh
$env:GO111MODULE = "on"
go build main.go
```

**Linux**

```sh
export GO111MODULE=on
go build main.go
```

## How to use it

**Note:** If you have built the project, then replace "go run main.go" with "main.exe" (for Windows) or "main" (for Linux/MacOS)

### Subscribe to all measurements for device id 12345 for 60 seconds

```sh
go run main.go -device 12345 -duration 60
```

### Subscribe to writeMinimum measurement series for device id 12345 for 60 seconds

```sh
go run main.go -device 12345 -duration 60 -series writeMinimum
```

### Subscribe to writeMinimum measurement series for device id 12345 for 60 seconds

```sh
go run main.go -device 12345 -duration 60 -series writeMinimum
```

### Subscribe to all measurements on all devices for 60 seconds

```sh
go run main.go -device * -duration 60 -series writeMinimum
```

### Subscribe to all operations on all devices for 60 seconds

```sh
go run main.go -device * -channel operations -duration 60
```
