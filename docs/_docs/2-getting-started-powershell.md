---
layout: default
category: Getting started
title: Powershell
---

Before getting started, you need to configure the Cumulocity session which detail which Cumulocity platform and authenication should be used for each of the commands/requests.

1. Import the PSc8y module into your Powershell console

    ```powershell
    Install-Module PSc8y -Repository PSGallery
    Import-Module PSc8y
    ```

1. Create a new session

    ```sh
    New-Session -Host "http://cumulocity.com" -Tenant "mytenant" -Username "myuser@me.com"
    ```

    You will be prompted for your password. Alternatively you can also enter the password using the `-Password` parameter.

    You may also provide a more meaningful session name by specifying a `-Name` argument.

1. Activate the session using the interactive session selector

    ```sh
    Set-Session
    ```

1. Test your credentials by getting your current user information from the platform

    ```sh
    Get-CurrentUser
    ```

    **Note**

    If your credentials are incorrect, then you can update the session file stored in the `~/.cumulocity` directory

1. Now you're ready to go. You can get a list of available commands by using help menu

    ```sh
    # List commands
    Get-Command -Module PSc8y

    # Get help for a command
    Get-Help Get-DeviceCollection -Full
    ```

### Switching sessions

The sessions can be changed again by using the interactive session selector

```sh
Set-Session
```
