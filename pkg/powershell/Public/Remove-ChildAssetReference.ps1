# Code generated from specification version 1.0.0: DO NOT EDIT
Function Remove-ChildAssetReference {
<#
.SYNOPSIS
Delete child asset reference


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Asset id (required)
        [Parameter(Mandatory = $true,
                   ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Asset,

        # Child device
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $ChildDevice,

        # Child device group
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $ChildGroup,

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
        if ($PSBoundParameters.ContainsKey("ChildDevice")) {
            $Parameters["childDevice"] = $ChildDevice
        }
        if ($PSBoundParameters.ContainsKey("ChildGroup")) {
            $Parameters["childGroup"] = $ChildGroup
        }
        if ($PSBoundParameters.ContainsKey("Session")) {
            $Parameters["session"] = $Session
        }

    }

    Process {
        foreach ($item in (PSC8y\Expand-Device $Asset)) {
            if ($item) {
                $Parameters["asset"] = if ($item.id) { $item.id } else { $item }
            }

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "inventoryReferences" `
                -Verb "deleteChildAsset" `
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
