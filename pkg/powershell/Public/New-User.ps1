# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-User {
<#
.SYNOPSIS
Create a new user within the collection


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

        # User name, unique for a given domain. Max: 1000 characters (required)
        [Parameter(Mandatory = $true)]
        [string]
        $UserName,

        # User first name
        [Parameter()]
        [string]
        $FirstName,

        # User last name
        [Parameter()]
        [string]
        $LastName,

        # User phone number. Format: '+[country code][number]', has to be a valid MSISDN
        [Parameter()]
        [string]
        $Phone,

        # User email address
        [Parameter()]
        [string]
        $Email,

        # User activation status (true/false)
        [Parameter()]
        [switch]
        $Enabled,

        # User password. Min: 6, max: 32 characters. Only Latin1 chars allowed (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Password,

        # User activation status (true/false)
        [Parameter()]
        [switch]
        $SendPasswordResetEmail,

        # User password. Min: 6, max: 32 characters. Only Latin1 chars allowed
        [Parameter()]
        [hashtable]
        $CustomProperties,

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
        if ($PSBoundParameters.ContainsKey("UserName")) {
            $Parameters["userName"] = $UserName
        }
        if ($PSBoundParameters.ContainsKey("FirstName")) {
            $Parameters["firstName"] = $FirstName
        }
        if ($PSBoundParameters.ContainsKey("LastName")) {
            $Parameters["lastName"] = $LastName
        }
        if ($PSBoundParameters.ContainsKey("Phone")) {
            $Parameters["phone"] = $Phone
        }
        if ($PSBoundParameters.ContainsKey("Email")) {
            $Parameters["email"] = $Email
        }
        if ($PSBoundParameters.ContainsKey("Enabled")) {
            $Parameters["enabled"] = $Enabled
        }
        if ($PSBoundParameters.ContainsKey("Password")) {
            $Parameters["password"] = $Password
        }
        if ($PSBoundParameters.ContainsKey("SendPasswordResetEmail")) {
            $Parameters["sendPasswordResetEmail"] = $SendPasswordResetEmail
        }
        if ($PSBoundParameters.ContainsKey("CustomProperties")) {
            $Parameters["customProperties"] = "{0}" -f ((ConvertTo-Json $CustomProperties -Compress) -replace '"', '\"')
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
                -Verb "create" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.user+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
