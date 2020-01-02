# Popular commands

## Basics

### Get a list of devices

```powershell
Get-DeviceCollection
```

### Display detailed logging about the command

```powershell
$resp = Get-DeviceCollection -Verbose
```

### Display the command to be sent to the Cumulocity platform without actually sending the command.

Create a new managed object

```powershell
New-ManagedObject -Name "settings_application_1" -Data @{ custom_value = "h3ll0"; } -WhatIf
```

### Subscribe to all realtime data for a given device

```powershell
Watch-Notifications -Device 12345
```

###