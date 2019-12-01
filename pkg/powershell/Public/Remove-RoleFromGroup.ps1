# Code generated from specification version 1.0.0: DO NOT EDIT
Function Remove-RoleFromGroup {
<#
.SYNOPSIS
Unassign/Remove role from a group


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Tenant
        [Parameter()]
        [object]
        $Tenant,

        # Group id (required)
        [Parameter(Mandatory = $true)]
        [object[]]
        $Group,

        # Role name, e.g. ROLE_TENANT_MANAGEMENT_ADMIN (required)
        [Parameter(Mandatory = $true)]
        [object[]]
        $Role,

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
        if ($PSBoundParameters.ContainsKey("Tenant")) {
            $Parameters["tenant"] = $Tenant
        }
        if ($PSBoundParameters.ContainsKey("Group")) {
            $Parameters["group"] = PSC8y\Expand-Id $Group
        }
        if ($PSBoundParameters.ContainsKey("Role")) {
            $Parameters["role"] = $Role
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
                -Noun "userRoles" `
                -Verb "deleteRoleFromGroup" `
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
