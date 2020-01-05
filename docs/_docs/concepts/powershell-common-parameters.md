---
layout: default
category: Concepts
title: Powershell - Common Parameters
---

# General parameters

## Verbose

### Display detailed logging about the command

```powershell
$resp = Get-DeviceCollection -Verbose
```

## Whatif

### Display the command to be sent to the Cumulocity platform without actually sending the command.

Create a new managed object

```powershell
New-ManagedObject -Name "settings_application_1" -Data @{ custom_value = "h3ll0"; } -WhatIf
```

## PageSize

## WithTotalPages

In the `PSc8y` module, the `-WithTotalPages` switch also changes the view to easily view the page statistics to see how many total pages exist.

In order to get an accurate total of entities, it is recommended to use `-PageSize 1` along with `-WithTotalPages`. The `totalPages` property will then display the total number of entities.

```powershell
Get-AlarmCollection -PageSize 1 -WithTotalPages
```

```powershell

    self: https://mytenant.xxxxx.cumulocity.com/alarm/alarms?withTotalPages=true&pageSize=1&currentPage=1
    next: https://mytenant.xxxxx.cumulocity.com/alarm/alarms?withTotalPages=true&pageSize=1&currentPage=2
currentPage     pageSize        totalPages      alarms
-----------     --------        ----------      ------
1               1               44              {@{severity=MAJOR; creationTime=12/23/2019 18:58:46...
```

### Raw

```powershell
Get-AlarmCollection -PageSize 1 -Raw | tojson
```

```json
{
  "next": "https://goc8yci01.eu-latest.cumulocity.com/alarm/alarms?pageSize=1&currentPage=2",
  "self": "https://goc8yci01.eu-latest.cumulocity.com/alarm/alarms?pageSize=1&currentPage=1",
  "statistics": {
    "totalPages": 44,
    "currentPage": 1,
    "pageSize": 1
  },
  "alarms": [
    {
      "severity": "MAJOR",
      "creationTime": "2019-12-23T18:58:46.069Z",
      "count": 1,
      "history": "@{auditRecords=System.Object[]; self=https://goc8yci01.eu-latest.cumulocity.com/audit/auditRecords}",
      "source": "@{name=TestDeviceuhEadfhHbT; self=https://goc8yci01.eu-latest.cumulocity.com/inventory/managedObjects/37571; id=37571}",
      "type": "testType",
      "self": "https://goc8yci01.eu-latest.cumulocity.com/alarm/alarms/37705",
      "time": "2019-12-23T18:58:45.771Z",
      "text": "Custom Event 1",
      "id": "37705",
      "status": "ACTIVE"
    }
  ]
}
```

Without the `-Raw` switch, the returned results, only the alarms are returned. This concept is applied to all entities (i.e. alarms, events. managed objecty, operations etc.)

```powershell
Get-AlarmCollection -PageSize 1 | tojson
```

**Response without `-Raw` switch**
```json
{
  "severity": "MAJOR",
  "creationTime": "2019-12-23T18:56:48.748Z",
  "count": 1,
  "history": {
    "auditRecords": [],
    "self": "https://goc8yci01.eu-latest.cumulocity.com/audit/auditRecords"
  },
  "source": {
    "name": "TestDeviceEGGWBKVMRa",
    "self": "https://goc8yci01.eu-latest.cumulocity.com/inventory/managedObjects/37512",
    "id": "37512"
  },
  "type": "testType",
  "self": "https://goc8yci01.eu-latest.cumulocity.com/alarm/alarms/37513",
  "time": "2019-12-23T18:56:48.325Z",
  "text": "Custom Event 1",
  "id": "37513",
  "status": "ACTIVE",
  "PSStatistics": {
    "currentPage": 1,
    "pageSize": 1,
    "totalPages": 44
  }
}
```

## OutputFile

The output file will be the raw response as returned from the Cumulocity platform.

```powershell
Get-AlarmCollection -PageSize 1 -OutputFile test.json
```

**Response**

```powershell
/Users/Shared/demo/test.json
```

Or it can be used to download a binary from the platform

c8y_applications_storage_9397

```powershell
Find-ManagedObjectCollection -Query "has(c8y_IsBinary) and type eq 'c8y_applications_storage_*'" -PageSize 1 |
    Get-Binary -OutputFile my.zip
```


## NoProxy

## WhatIf

## Session
