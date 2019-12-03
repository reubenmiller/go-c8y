# Automatic code generation

## Generating the code

```sh
./build.ps1

```

## Roadmap

* [x] Support common parameters
  * [x] pageSize
  * [x] withTotalPages
* [x] Add required parameters
* [x] Piped arguments
* [x] Set defaults for certain variables. i.e. Tenant
* [x] Commands
  * [x] Devices list --type unitType
* [x] Expansion
  * [x] applications
  * [x] devices
* [x] Flag parsing
* [x] Suppress logging when not in verbose mode
  * [x] Datetime (relative and fixed)
* [x] New / Import / export cumulocity sessions
  * [x] Create new session
  * [x] Import a session from file
* [x] Generate powershell commands from templates
* [x] Result parsing
  * [x] client side filtering. e.g. c8y applications list --filter "name=*test*"
* [x] Add response size to log
* [x] Support more filtering possibilities
  * [x] Wildcard
  * [x] Regex
* [ ] Adding timeout argument
* [ ] Add request response time to log
* [ ] Add examples
* [ ] Generate tests automatically
* [ ] Microservice aliases using my-app://health
* [ ] Add "file" argument type
* [ ] Review "set" argument type
* [x] Lookups
  * [x] Add role lookup, which converts a name to a self link. required for Add-RoleToUser
  * [x] Add user lookup
  * [x] Add user self reference lookup
  * [x] Add user group lookup
* [ ] Add outFile flag
  * [ ] Update all download files
* [ ] Add upload flag
  * [ ] Update all upload files
* [ ] Generic download file cmd
* [ ] Generic upload file cmd
* [ ] Generic rest cmd
  * [ ] If the response is not json, then return it as is (i.e. like the --raw switch)

### Phase 2

* [ ] Make options case insensitive
* [ ] Look over devices where []device type is used (parallel tasks?) Probably need a new template
* [ ] Cumulocity sessions
  * [ ] Store session credentials securely
  * [ ] Set credentials from a microservice subscription



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
* [x] Support for ShouldProcess prompt
  * [x] Support device name lookup in the message?
* [ ] ?native multi-part upload?
* [ ] Add tests
  * [ ] How to automatic generate Pester tests
* [ ] Return status codes
* [x] Use session default values (C8Y_TENANT for tenant path/query variables)

* [ ] Remove child devices and child references by wildcard. Only delete matching children

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



## Controlling the active session

## Option 1: Setting the C8Y_HOST, C8Y_USER, C8Y_PASSWORD, C8Y_TENANT env variables

**Disadvantages**
* Use must set these every time, this is very tedious
* Setting the variables is different for each OS

**Advantages**
* Simple
* If these env variables are already set, then there is nothing else to do
* The user is responsible for setting these themselves

## Option 2: Multiple session files

* Keep one session per file, and set one environment variable which points to the "active" session?
* autocomplete with files?

Example: **c8y.myfilter.session**
* One file stores the default sessions
*

## Option 2:

# Troubleshooting

## bash completion does not work

**Description**

After generating the bash completions

```sh
c8y completions bash > .c8y.sh
source .c8y.sh
```
The following error is displayed:

```sh
bash: _get_comp_words_by_ref: command not found warning
```

**Fix**
Install bash-completions

```sh
yum install bash-completion bash-completion-extras
```

Note: You need to start a new bash session before the bash add-ons are activated
