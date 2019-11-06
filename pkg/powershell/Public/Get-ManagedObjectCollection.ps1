# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-ManagedObjectCollection {
<#
.SYNOPSIS
Get a collection of managedObjects based on filter parameters

.DESCRIPTION
Get a collection of managedObjects based on filter parameters

.EXAMPLE
Get-ManagedobjectCollection


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Source device id.
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Device,

        # ManagedObject type.
        [Parameter()]
        [string]
        $Type,

        # ManagedObject fragment type.
        [Parameter()]
        [string]
        $FragmentType,

        # managed objects containing a text value starting with the given text (placeholder {text}). Text value is any alphanumeric string starting with a latin letter (A-Z or a-z).
        [Parameter()]
        [string]
        $Text,

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
        if ($PSBoundParameters.ContainsKey("Device")) {
            $Parameters["device"] = $Device
        }
        if ($PSBoundParameters.ContainsKey("Type")) {
            $Parameters["type"] = $Type
        }
        if ($PSBoundParameters.ContainsKey("FragmentType")) {
            $Parameters["fragmentType"] = $FragmentType
        }
        if ($PSBoundParameters.ContainsKey("Text")) {
            $Parameters["text"] = $Text
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
                -Verb "list" `
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
