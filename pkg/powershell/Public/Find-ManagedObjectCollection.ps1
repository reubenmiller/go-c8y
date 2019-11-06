# Code generated from specification version 1.0.0: DO NOT EDIT
Function Find-ManagedObjectCollection {
<#
.SYNOPSIS
Get a collection of managedObjects based on Cumulocity query language

.DESCRIPTION
Get a collection of managedObjects based on Cumulocity query language

.EXAMPLE
Find-ManagedobjectCollection


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # ManagedObject query.
        [Parameter()]
        [string]
        $Query,

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
        if ($PSBoundParameters.ContainsKey("Query")) {
            $Parameters["query"] = $Query
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
                -Verb "query" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.managedObjectCollection+json" `
                -ItemType "application/vnd.com.nsn.cumulocity.managedObject+json" `
                -ResultProperty "managedObjects" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
