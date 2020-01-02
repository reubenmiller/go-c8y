# lookups

## Accessing devices by name

All devices which require a device id, can also be 

### Bash

```sh
c8ycli alarms list --device mydevice
```

### Powershell

```powershell
Get-AlarmCollection -Device mydevice
```

## Get application by name

### Bash

```sh
c8ycli applications get --id "cockpit
```

### Powershell

```powershell
Get-Application -Id "cockpit"
```
