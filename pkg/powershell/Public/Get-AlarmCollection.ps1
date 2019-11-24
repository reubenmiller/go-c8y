# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-AlarmCollection {
<#
.SYNOPSIS
Get a collection of alarms based on filter parameters

.DESCRIPTION
Get a collection of alarms based on filter parameters

.EXAMPLE
PS> Get-AlarmCollection -Severity MAJOR -PageSize 100
Get alarms with the severity set to MAJOR

.EXAMPLE
PS> Get-AlarmCollection -DateFrom "-10m" -Status ACTIVE
Get active alarms which occurred in the last 10 minutes


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Source device id.
        [Parameter()]
        [object[]]
        $Device,

        # Start date or date and time of alarm occurrence.
        [Parameter()]
        [string]
        $DateFrom,

        # End date or date and time of alarm occurrence.
        [Parameter()]
        [string]
        $DateTo,

        # Alarm type.
        [Parameter()]
        [string]
        $Type,

        # Alarm fragment type.
        [Parameter()]
        [string]
        $FragmentType,

        # Comma separated alarm statuses, for example ACTIVE,CLEARED.
        [Parameter()]
        [ValidateSet('ACTIVE','ACKNOWLEDGED','CLEARED')]
        [string]
        $Status,

        # Alarm severity, for example CRITICAL, MAJOR, MINOR or WARNING.
        [Parameter()]
        [ValidateSet('CRITICAL','MAJOR','MINOR','WARNING')]
        [string]
        $Severity,

        # When set to true only resolved alarms will be removed (the one with status CLEARED), false means alarms with status ACTIVE or ACKNOWLEDGED.
        [Parameter()]
        [switch]
        $Resolved,

        # Include assets
        [Parameter()]
        [switch]
        $WithAssets,

        # Include devices
        [Parameter()]
        [switch]
        $WithDevices,

        # Maximum number of results
        [Parameter()]
        [AllowNull()]
        [AllowEmptyString()]
        [ValidateRange(1,2000)]
        [int]
        $PageSize,

        # Include total pages statistic
        [Parameter()]
        [switch]
        $WithTotalPages,

        # Include all results
        [Parameter()]
        [switch]
        $IncludeAll,

        # Include raw response including pagination information
        [Parameter()]
        [switch]
        $Raw,

        # Session path
        [Parameter()]
        [string]
        $Session
    )

    Begin {
        $Parameters = @{}
        if ($PSBoundParameters.ContainsKey("Device")) {
            $Parameters["device"] = $Device
        }
        if ($PSBoundParameters.ContainsKey("DateFrom")) {
            $Parameters["dateFrom"] = $DateFrom
        }
        if ($PSBoundParameters.ContainsKey("DateTo")) {
            $Parameters["dateTo"] = $DateTo
        }
        if ($PSBoundParameters.ContainsKey("Type")) {
            $Parameters["type"] = $Type
        }
        if ($PSBoundParameters.ContainsKey("FragmentType")) {
            $Parameters["fragmentType"] = $FragmentType
        }
        if ($PSBoundParameters.ContainsKey("Status")) {
            $Parameters["status"] = $Status
        }
        if ($PSBoundParameters.ContainsKey("Severity")) {
            $Parameters["severity"] = $Severity
        }
        if ($PSBoundParameters.ContainsKey("Resolved")) {
            $Parameters["resolved"] = $Resolved
        }
        if ($PSBoundParameters.ContainsKey("WithAssets")) {
            $Parameters["withAssets"] = $WithAssets
        }
        if ($PSBoundParameters.ContainsKey("WithDevices")) {
            $Parameters["withDevices"] = $WithDevices
        }
        if ($PSBoundParameters.ContainsKey("PageSize")) {
            $Parameters["pageSize"] = $PageSize
        }
        if ($PSBoundParameters.ContainsKey("WithTotalPages") -and $WithTotalPages) {
            $Parameters["withTotalPages"] = $WithTotalPages
        }
        if ($PSBoundParameters.ContainsKey("Session")) {
            $Parameters["session"] = $Session
        }

    }

    Process {
        foreach ($item in @("")) {

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (PSC8y\Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "alarms" `
                -Verb "list" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.alarmCollection+json" `
                -ItemType "application/vnd.com.nsn.cumulocity.alarm+json" `
                -ResultProperty "alarms" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
