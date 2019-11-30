# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-UserCollection {
<#
.SYNOPSIS
Get a collection of users based on filter parameters

.DESCRIPTION
Get a collection of users based on filter parameters

.EXAMPLE
PS> Get-UserCollection
Get a list of users


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Tenant
        [Parameter()]
        [object]
        $Tenant,

        # prefix or full username
        [Parameter()]
        [string]
        $Username,

        # numeric group identifiers separated by commas; result will contain only users which belong to at least one of specified groups
        [Parameter()]
        [string]
        $Groups,

        # exact username
        [Parameter()]
        [string]
        $Owner,

        # If set to 'true', result will contain only users created during bootstrap process (starting with 'device_'). If flag is absent (or false) the result will not contain 'device_' users.
        [Parameter()]
        [switch]
        $OnlyDevices,

        # if set to 'true', then each of returned users will contain additional field 'subusersCount' - number of direct subusers (users with corresponding 'owner').
        [Parameter()]
        [switch]
        $WithSubusersCount,

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
        if ($PSBoundParameters.ContainsKey("Tenant")) {
            $Parameters["tenant"] = $Tenant
        }
        if ($PSBoundParameters.ContainsKey("Username")) {
            $Parameters["username"] = $Username
        }
        if ($PSBoundParameters.ContainsKey("Groups")) {
            $Parameters["groups"] = $Groups
        }
        if ($PSBoundParameters.ContainsKey("Owner")) {
            $Parameters["owner"] = $Owner
        }
        if ($PSBoundParameters.ContainsKey("OnlyDevices")) {
            $Parameters["onlyDevices"] = $OnlyDevices
        }
        if ($PSBoundParameters.ContainsKey("WithSubusersCount")) {
            $Parameters["withSubusersCount"] = $WithSubusersCount
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
                    (PSC8y\Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "users" `
                -Verb "list" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.userCollection+json" `
                -ItemType "application/vnd.com.nsn.cumulocity.user+json" `
                -ResultProperty "users" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
