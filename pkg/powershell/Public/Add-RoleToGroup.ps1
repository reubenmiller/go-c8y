# Code generated from specification version 1.0.0: DO NOT EDIT
Function Add-RoleToGroup {
<#
.SYNOPSIS
Add role to a group


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

        # Group ID (required)
        [Parameter(Mandatory = $true)]
        [string]
        $GroupId,

        # User role id (required)
        [Parameter(Mandatory = $true)]
        [string]
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
        if ($PSBoundParameters.ContainsKey("GroupId")) {
            $Parameters["groupId"] = $GroupId
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
                -Verb "addRoleToGroup" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.roleReference+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
