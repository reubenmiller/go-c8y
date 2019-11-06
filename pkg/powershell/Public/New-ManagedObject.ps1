# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-ManagedObject {
<#
.SYNOPSIS
Create a new inventory

.DESCRIPTION
Create a new inventory


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # The ManagedObject that the inventory originated from (required)
        [Parameter(Mandatory = $true,
                   ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Device,

        # Identifies the type of this inventory, e.g. 'com_cumulocity_events_TamperEvent'. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Type,

        # Time of the inventory. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Time,

        # Text description of the inventory. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Text,

        # The severity of the inventory: CRITICAL, MAJOR, MINOR or WARNING. Must be upper-case. (required)
        [Parameter(Mandatory = $true)]
        [ValidateSet('CRITICAL','MAJOR','MINOR','WARNING')]
        [string]
        $Severity,

        # The status of the inventory: ACTIVE, ACKNOWLEDGED or CLEARED. If status was not appeared, new inventory will have status ACTIVE. Must be upper-case.
        [Parameter()]
        [ValidateSet('ACTIVE','ACKNOWLEDGED','CLEARED')]
        [string]
        $Status,

        # Additional properties of the inventory.
        [Parameter()]
        [hashtable]
        $Data,

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
        if ($PSBoundParameters.ContainsKey("Type")) {
            $Parameters["type"] = $Type
        }
        if ($PSBoundParameters.ContainsKey("Time")) {
            $Parameters["time"] = $Time
        }
        if ($PSBoundParameters.ContainsKey("Text")) {
            $Parameters["text"] = $Text
        }
        if ($PSBoundParameters.ContainsKey("Severity")) {
            $Parameters["severity"] = $Severity
        }
        if ($PSBoundParameters.ContainsKey("Status")) {
            $Parameters["status"] = $Status
        }
        if ($PSBoundParameters.ContainsKey("Data")) {
            $Parameters["data"] = "{0}" -f ((ConvertTo-Json $Data -Compress) -replace '"', '\"')
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
                    (Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "inventory" `
                -Verb "create" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.inventory+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
