# Code generated from specification version 1.0.0: DO NOT EDIT
Function Remove-AlarmCollection {
<#
.SYNOPSIS
Delete a collection of alarms

.EXAMPLE
PS> Remove-AlarmCollection -Device "{{ randomdevice }}" -Severity MAJOR
Remove alarms on the device with the severity set to MAJOR

.EXAMPLE
PS> Remove-AlarmCollection -Device "{{ randomdevice }}" -DateFrom "-10m" -Status ACTIVE
Remove alarms on the device which are active and created in the last 10 minutes


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
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

        # When set to true also alarms for related source assets will be removed. When this parameter is provided also source must be defined.
        [Parameter()]
        [switch]
        $WithSourceAssets,

        # When set to true also alarms for related source devices will be removed. When this parameter is provided also source must be defined.
        [Parameter()]
        [switch]
        $WithSourceDevices,

        # Include raw response including pagination information
        [Parameter()]
        [switch]
        $Raw,

        # Session path
        [Parameter()]
        [string]
        $Session,

        # Don't prompt for confirmation
        [Parameter()]
        [switch]
        $Force
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
        if ($PSBoundParameters.ContainsKey("WithSourceAssets")) {
            $Parameters["withSourceAssets"] = $WithSourceAssets
        }
        if ($PSBoundParameters.ContainsKey("WithSourceDevices")) {
            $Parameters["withSourceDevices"] = $WithSourceDevices
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
                -Verb "deleteCollection" `
                -Parameters $Parameters `
                -Type "" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
