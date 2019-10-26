# Code generated from specification version 1.0.0: DO NOT EDIT
Function Update-AlarmCollection {
<#
.SYNOPSIS
Update a collection of alarms. Currently only the status of alarms can be changed


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # The ManagedObject that the alarm originated from
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
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
        [datetime]
        $DateFrom,

        # End date or date and time of alarm occurrence.
        [Parameter()]
        [datetime]
        $DateTo,

        # New status to be applied to all of the matching alarms (required)
        [Parameter(Mandatory = $true)]
        [ValidateSet('ACTIVE','ACKNOWLEDGED','CLEARED')]
        [string]
        $NewStatus,

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
        $Raw
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
            $Parameters["dateFrom"] = PSC8y\Format-Date $DateFrom
        }
        if ($PSBoundParameters.ContainsKey("DateTo")) {
            $Parameters["dateTo"] = PSC8y\Format-Date $DateTo
        }
        if ($PSBoundParameters.ContainsKey("NewStatus")) {
            $Parameters["newStatus"] = $NewStatus
        }

    }

    Process {
        foreach ($item in @("")) {

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "alarms" `
                -Verb "updateCollection" `
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
