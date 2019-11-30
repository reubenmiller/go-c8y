# Code generated from specification version 1.0.0: DO NOT EDIT
Function Update-AlarmCollection {
<#
.SYNOPSIS
Update a collection of alarms. Currently only the status of alarms can be changed

.EXAMPLE
PS> Update-AlarmCollection -Device $Device.id -Status ACTIVE -NewStatus ACKNOWLEDGED
Update the status of all active alarms on a device to ACKNOWLEDGED


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # The ManagedObject that the alarm originated from
        [Parameter()]
        [object[]]
        $Device,

        # The status of the alarm: ACTIVE, ACKNOWLEDGED or CLEARED. If status was not appeared, new alarm will have status ACTIVE. Must be upper-case.
        [Parameter()]
        [ValidateSet('ACTIVE','ACKNOWLEDGED','CLEARED')]
        [string]
        $Status,

        # The severity of the alarm: CRITICAL, MAJOR, MINOR or WARNING. Must be upper-case.
        [Parameter()]
        [ValidateSet('CRITICAL','MAJOR','MINOR','WARNING')]
        [string]
        $Severity,

        # When set to true only resolved alarms will be removed (the one with status CLEARED), false means alarms with status ACTIVE or ACKNOWLEDGED.
        [Parameter()]
        [switch]
        $Resolved,

        # Start date or date and time of alarm occurrence.
        [Parameter()]
        [string]
        $DateFrom,

        # End date or date and time of alarm occurrence.
        [Parameter()]
        [string]
        $DateTo,

        # New status to be applied to all of the matching alarms (required)
        [Parameter(Mandatory = $true)]
        [ValidateSet('ACTIVE','ACKNOWLEDGED','CLEARED')]
        [string]
        $NewStatus,

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
        if ($PSBoundParameters.ContainsKey("Status")) {
            $Parameters["status"] = $Status
        }
        if ($PSBoundParameters.ContainsKey("Severity")) {
            $Parameters["severity"] = $Severity
        }
        if ($PSBoundParameters.ContainsKey("Resolved")) {
            $Parameters["resolved"] = $Resolved
        }
        if ($PSBoundParameters.ContainsKey("DateFrom")) {
            $Parameters["dateFrom"] = $DateFrom
        }
        if ($PSBoundParameters.ContainsKey("DateTo")) {
            $Parameters["dateTo"] = $DateTo
        }
        if ($PSBoundParameters.ContainsKey("NewStatus")) {
            $Parameters["newStatus"] = $NewStatus
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
                -Verb "updateCollection" `
                -Parameters $Parameters `
                -Type "" `
                -ItemType "" `
                -ResultProperty "alarms" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
