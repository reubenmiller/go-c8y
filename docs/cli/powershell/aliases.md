# Aliases

In order to make common commands easier/quicker to use, a few default aliases can be registered using:

```powershell
Register-Alias
```

These same aliases can be unregistered using:

```powershell
Unregister-Alias
```

Below is a list of the aliases:


* `alarm` -> `Get-Alarm`
* `alarms` -> `Get-AlarmCollection`
* `app` -> `Get-Application`
* `apps` -> `Get-ApplicationCollection`
* `devices` -> `Get-DeviceCollection`
* `event` -> `Get-Event`
* `events` -> `Get-EventCollection`
* `fmo` -> `Find-ManagedObjectCollection`
* `fromjson` -> `ConvertFrom-Json`
* `json` -> `ConvertTo-Json`
* `m` -> `Get-Measurements`
* `measurements` -> `Get-MeasurementCollection`
* `mo` -> `Get-ManagedObject`
* `op` -> `Get-Operation`
* `ops` -> `Get-OperationCollection`
* `rest` -> `Invoke-RestRequest`
* `series` -> `Get-MeasurementSeries`
* `session` -> `Get-Session`
* `tojson` -> `ConvertTo-Json`

Custom Alias can still be registered by the user using

