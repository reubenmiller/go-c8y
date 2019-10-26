# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-AuditRecordCollection {
<#
.SYNOPSIS
Get collection of (user) audits


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Source Id or object containing an .id property of the element that should be detected. i.e. AlarmID, or Operation ID. Note: Only one source can be provided
        [Parameter()]
        [string]
        $Source,

        # Type
        [Parameter()]
        [string]
        $Type,

        # Username
        [Parameter()]
        [string]
        $User,

        # Application
        [Parameter()]
        [string]
        $Application,

        # Start date or date and time of audit record occurrence.
        [Parameter()]
        [datetime]
        $DateFrom,

        # End date or date and time of audit record occurrence.
        [Parameter()]
        [datetime]
        $DateTo,

        # Return the newest instead of the oldest audit records. Must be used with dateFrom and dateTo parameters
        [Parameter()]
        [switch]
        $Revert,

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
        if ($PSBoundParameters.ContainsKey("Source")) {
            $Parameters["source"] = $Source
        }
        if ($PSBoundParameters.ContainsKey("Type")) {
            $Parameters["type"] = $Type
        }
        if ($PSBoundParameters.ContainsKey("User")) {
            $Parameters["user"] = $User
        }
        if ($PSBoundParameters.ContainsKey("Application")) {
            $Parameters["application"] = $Application
        }
        if ($PSBoundParameters.ContainsKey("DateFrom")) {
            $Parameters["dateFrom"] = PSC8y\Format-Date $DateFrom
        }
        if ($PSBoundParameters.ContainsKey("DateTo")) {
            $Parameters["dateTo"] = PSC8y\Format-Date $DateTo
        }
        if ($PSBoundParameters.ContainsKey("Revert")) {
            $Parameters["revert"] = $Revert
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
                -Noun "auditRecords" `
                -Verb "list" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.auditRecordCollection+json" `
                -ItemType "application/vnd.com.nsn.cumulocity.auditRecord+json" `
                -ResultProperty "auditRecords" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
