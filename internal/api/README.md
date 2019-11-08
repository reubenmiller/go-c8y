# Automatic code generation

## Generating the code

```sh
./build.ps1

```

## Roadmap

* [ ] Support common parameters
  * [x] pageSize
  * [x] withTotalPages
* [x] Add required parameters
* [ ] Piped arguments
* [ ] Look over devices where []device type is used (parallel tasks?) Probably need a new
* [ ] Commands
  * [ ] Devices list --type unitType
* [ ] Expansion
  * [x] applications
  * [ ] devices
  * [ ] agents
* [ ] Result parsing
  * [ ] client side filtering. e.g. c8y applications list --filter "name=*test*"
* [ ] New / Import / export cumulocity sessions
  * [ ] Generate secure password and encrypt it
  * [ ] Set credentials from a microservice subscription
  * [ ] encrypt/decrypt password
* [ ] Create new session
* [x] Flag parsing
  * [x] Datetime (relative and fixed)
* [ ] template
* [ ] Add examples
* [ ] Generate tests automatically
* [x] Generate powershell commands from templates
* [ ] Make options case insensitive

# encryption process

1. Generate a unique token, store it in an environment variable
2.

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



## Controlling the active session

## Option 1: Setting the C8Y_HOST, C8Y_USER, C8Y_PASSWORD, C8Y_TENANT env variables

**Disadvantsges**
* Use must set these every time, this is very tedious
* Setting the variables is different for each OS

**Advantages**
* Simple
* If these env variables are already set, then there is nothing else to do
* The user is reponsible for setting these themselves

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
