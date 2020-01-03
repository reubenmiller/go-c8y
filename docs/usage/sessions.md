# Sessions

The active session is controlled either by `C8Y_SESSION` environmnet variable. The environment variable should be pointed to the JSON file.

```sh
# Bash
export C8Y_SESSION=~/.cumulocity/my-settings01.json

# Powershell
$env:C8Y_SESSION = "~/.cumulocity/my-settings01.json"
```

## Setting settings via Environmnet variables (Continueous Integration usage)

Alternatively, the Cumulocity settings used by the c8ycli can be controlled purely by environment variables (ideally suited for Continuous Integration)

Firstly, the `C8Y_USE_ENVIRONMENT` environment needs to be set to `true` to activate this mode.

Then the Cumulocity settings can be set by the following environment variables.

* C8Y_HOST (example "https://cumulocity.com")
* C8Y_TENANT (example "mytest")
* C8Y_USER
* C8Y_PASSWORD

## Switch sessions 

### Bash

The sessions can be changed again by using the interactive session selector

```sh
export C8Y_SESSION=$( ./c8ycli sessions list )
```

### Powershell

```sh
Set-Session
```

## Switching session for a single command

If you only need to set a session for a single session, then you can use the global `--session` argument. The name of the session should be the name of the file stored under your `~/.cumulocity/` folder (with or without the .json extension).

You can set the `C8Y_SESSION_HOME` environment variable to control where the sessions should be stored.

### Bash

```sh
c8ycli devices list --session myother.tenant
```

### Powershell

```sh
Get-DeviceCollection -Session myother.tenant
```
