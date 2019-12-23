Function Get-DeviceCollection {
<#
.SYNOPSIS
Get the active Cumulocity Session

.EXAMPLE
Get-Session

Get the current Cumulocity session

#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Device name.
        [Parameter(Mandatory = $false)]
        [string]
        $Name,

        # Device type.
        [Parameter(Mandatory = $false)]
        [string]
        $Type,

        # Device fragment type.
        [Parameter(Mandatory = $false)]
        [string]
        $FragmentType,

        # Device owner.
        [Parameter(Mandatory = $false)]
        [string]
        $Owner,

        # Query.
        [Parameter(Mandatory = $false)]
        [string]
        $Query,

        # Only include agents.
        [Parameter()]
        [switch]
        $Agents,

        # include a flat list of all parents and grandparents of the given object
        [Parameter()]
        [switch]
        $WithParents,

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
        if ($PSBoundParameters.ContainsKey("Name")) {
            $Parameters["name"] = $Name
        }
        if ($PSBoundParameters.ContainsKey("Type")) {
            $Parameters["type"] = $Type
        }
        if ($PSBoundParameters.ContainsKey("FragmentType")) {
            $Parameters["fragmentType"] = $FragmentType
        }
        if ($PSBoundParameters.ContainsKey("owner")) {
            $Parameters["owner"] = $Owner
        }
        if ($PSBoundParameters.ContainsKey("Query")) {
            $Parameters["query"] = $Query
        }
        if ($PSBoundParameters.ContainsKey("Agents")) {
            $Parameters["agents"] = $Agents
        }
        if ($PSBoundParameters.ContainsKey("WithParents")) {
            $Parameters["withParents"] = $WithParents
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
        if (!$Force -and
            !$WhatIfPreference -and
            !$PSCmdlet.ShouldProcess(
                (PSc8y\Get-C8ySessionProperty -Name "tenant"),
                (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject "")
            )) {
            continue
        }

        Invoke-Command `
            -Noun "devices" `
            -Verb "list" `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.managedObjectCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.managedObject+json" `
            -ResultProperty "managedObjects" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

}
