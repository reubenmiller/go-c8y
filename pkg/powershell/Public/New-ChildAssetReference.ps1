# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-ChildAssetReference {
<#
.SYNOPSIS
Create a child asset (device or devicegroup) reference

.DESCRIPTION
Create a child asset (device or devicegroup) reference


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Group id (required)
        [Parameter(Mandatory = $true,
                   ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Group,

        # new child device asset
        [Parameter()]
        [object[]]
        $NewChildDevice,

        # new child device group asset
        [Parameter()]
        [object[]]
        $NewChildGroup,

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
        if ($PSBoundParameters.ContainsKey("NewChildDevice")) {
            $Parameters["newChildDevice"] = $NewChildDevice
        }
        if ($PSBoundParameters.ContainsKey("NewChildGroup")) {
            $Parameters["newChildGroup"] = $NewChildGroup
        }
        if ($PSBoundParameters.ContainsKey("Session")) {
            $Parameters["session"] = $Session
        }

    }

    Process {
        foreach ($item in (PSC8y\Expand-Id $Group)) {
            if ($item) {
                $Parameters["group"] = if ($item.id) { $item.id } else { $item }
            }

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (PSC8y\Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "inventoryReferences" `
                -Verb "createChildAsset" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.managedObjectReference+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
