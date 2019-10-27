# Automatic code generation

## Generating the code

```sh
./build.ps1

```

## Roadmap

* [ ] Support common parameters
  * [x] pageSize
  * [x] withTotalPages
  * [ ] --raw Option to display raw result, rather than a filtered data set?
* [x] Add required parameters
* [ ] Piped arguments
* [ ] Look over devices where []device type is used (parallel tasks?) Probably need a new
* [ ] Commands
  * [ ] Devices list --type unitType
* [ ] Expansion
  * [ ] applications
* [ ] template
* [ ] Value formatter (for self link values), or is this like the device type?
* [ ] Add examples
* [ ] Generate tests automatically
* [ ] Generate powershell commands from templates
* [ ] Make options case insensitive

## Powershell

* [x] Support common parameters
  * [x] PageSize
  * [x] WithTotalPages
  * [x] Raw
  * [x] Force
  * [ ] Without Accept header (for performance improvements)
* [x] Validate set
* [x] Add types (using cumulocity types) and default columns
  * [x] Get-AlarmCollection.ps1
  * [x] Get-ApplicationCollection.ps1
  * [x] Get-ApplicationReferenceCollection.ps1
  * [x] Get-AuditRecordCollection.ps1
  * [x] Get-BinaryCollection.ps1
  * [x] Get-EventCollection.ps1
  * [x] Get-ExternalIDCollection.ps1
  * [x] Get-GroupCollection.ps1
  * [x] Get-MeasurementCollection.ps1
  * [x] Get-OperationCollection.ps1
  * [x] Get-RetentionRuleCollection.ps1
  * [x] Get-RoleReferenceCollectionFromGroup.ps1
  * [x] Get-RoleReferenceCollectionFromUser.ps1
  * [x] Get-SystemOptionCollection.ps1
  * [x] Get-TenantCollection.ps1
  * [x] Get-TenantOptionCollection.ps1
  * [x] Get-TenantStatisticsCollection.ps1
  * [x] Get-UserCollection.ps1
  * [x] Update-AlarmCollection.ps1
* [ ] Parameter types
  * [ ] File
  * [ ] Data
    * [x] hashtable
    * [ ] manual json or json shortform
  * [ ] device expansion (if given an id, don't do a lookup)
  * [x] application
* [ ] Client side filtering of results for those that don't support server side filters
  * [ ] Application
    * [ ] Name
    * [ ] type
  * [ ]
* [ ] Support for ShouldProcess prompt
  * [ ] Support device name lookup in the message?
* [ ] ?native multi-part upload?
* [ ] Add tests
  * [ ] How to automatic generate Pester tests
* [ ] Return status codes
* [ ] Use session default values (C8Y_TENANT for tenant path/query variables)


Manual commands

* [ ] applications
  * [ ] New-Microservice
  * [ ]


## Command layout

- applications
    - list
    - get
    - create
    - copy
    - delete
    - createBinary
    - getBootstrapUser

- retentionRules
    - list
    - create
    - get
    - delete
    - update

- systemOptions
    - list
    - get

- binaries
    - create
    - delete
    - download
    - list
    - update

- alarms
    - delete
    - deleteCollection
    - get
    - list
    - new
    - update
    - updateCollection

- tenants
    - list
    - create
    - get
    - delete
    - update
    - getCurrentTenant
    - enableApplication
    - disableApplication
    - listApplicationReferences
    -
- tenantOptions
    - list
    - create
    - get
    - delete
    - update
    - updateBulk
    - getForCategory
    - updateEdit
    - updateEdit
    -

- tenantStatistics
    - list
    - listSummaryAllTenants
    - listSummaryForTenant

- currentApplication
    - get
    - update
    - listSubscriptions

-
