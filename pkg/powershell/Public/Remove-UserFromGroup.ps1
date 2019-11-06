# Code generated from specification version 1.0.0: DO NOT EDIT
Function Remove-UserFromGroup {
<#
.SYNOPSIS
Delete a user from a group


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
        [string]
        $Tenant,

        # Group ID
        [Parameter()]
        [string]
        $GroupId,

        # User id/username
        [Parameter()]
        [string]
        $UserId,

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
        if ($PSBoundParameters.ContainsKey("UserId")) {
            $Parameters["userId"] = $UserId
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
                -Noun "userReferences" `
                -Verb "deleteUserFromGroup" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.currentUser+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
