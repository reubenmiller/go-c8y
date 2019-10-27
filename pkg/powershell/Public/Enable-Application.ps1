# Code generated from specification version 1.0.0: DO NOT EDIT
Function Enable-Application {
<#
.SYNOPSIS
Enable application on tenant


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Tenant id (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Tenant,

        # Application id (required)
        [Parameter(Mandatory = $true,
                   ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Application,

        # Include raw response including pagination information
        [Parameter()]
        [switch]
        $Raw,

        # Don't prompt for confirmation
        [Parameter()]
        [switch]
        $Force
    )

    Begin {
        $Parameters = @{}
        if ($PSBoundParameters.ContainsKey("Tenant")) {
            $Parameters["tenant"] = $Tenant
        }

    }

    Process {
        foreach ($item in (PSC8y\Expand-Application $Application)) {
            if ($item) {
                $Parameters["application"] = if ($item.id) { $item.id } else { $item }
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
                -Noun "tenants" `
                -Verb "enableApplication" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.applicationReference+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
