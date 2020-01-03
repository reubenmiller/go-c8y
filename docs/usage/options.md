# Options

c8y and PSc8y can be configured using environment variables to control certain behaviours. The following is a list of available options.

### C8Y_SESSION

Path to the Cumulocity session file to be used.

If it exists when the PSc8y PowerShell module is loaded, then the session will be loaded automatically.

### C8Y_USE_ENVIRONMENT

When set to `on`, the Cumulocity session settings will be loaded from the following environment variables:

* C8Y_HOST (example "https://cumulocity.com")
* C8Y_TENANT (example "mytest")
* C8Y_USER
* C8Y_PASSWORD

`C8Y_USE_ENVIRONMENT` will override the `C8Y_SESSION` environment variable.

```sh
# bash
export C8Y_USE_ENVIRONMENT=on

# PowerShell
$env:C8Y_USE_ENVIRONMENT = "on"
```

### C8Y_SESSION_HOME

By default the `$HOME/.cumulocity` directory is used to store the Cumulocity session files. A custom session home folder can be specified by setting the `C8Y_SESSION_HOME` to a folder.
Use a custom folder where the Cumulocity Session files should be keep and scanned. 

### PSC8Y_SKIP_INSTALL_ON_IMPORT (PowerShell only)

On MacOS and Linux don't install the c8y executable when loading the PSc8y PowerShell module.

Example:

```PowerShell
export PSC8Y_SKIP_INSTALL_ON_IMPORT=on
```
