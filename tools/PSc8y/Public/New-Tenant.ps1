# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-Tenant {
<#
.SYNOPSIS
New tenant


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Company name. Maximum 256 characters (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Company,

        # Domain name to be used for the tenant. Maximum 256 characters (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Domain,

        # The tenant ID. Will be auto-generated if not present.
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [string]
        $Id,

        # Username of the tenant administrator
        [Parameter()]
        [string]
        $AdminName,

        # Password of the tenant administrator
        [Parameter()]
        [string]
        $AdminPass,

        # A contact name, for example an administrator, of the tenant
        [Parameter()]
        [string]
        $ContactName,

        # An international contact phone number
        [Parameter()]
        [string]
        $Contact_phone,

        # A set of custom properties of the tenant
        [Parameter()]
        [hashtable]
        $Data,

        # Include raw response including pagination information
        [Parameter()]
        [switch]
        $Raw,

        # Outputfile
        [Parameter()]
        [string]
        $OutputFile,

        # NoProxy
        [Parameter()]
        [switch]
        $NoProxy,

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
        if ($PSBoundParameters.ContainsKey("Company")) {
            $Parameters["company"] = $Company
        }
        if ($PSBoundParameters.ContainsKey("Domain")) {
            $Parameters["domain"] = $Domain
        }
        if ($PSBoundParameters.ContainsKey("Id")) {
            $Parameters["id"] = $Id
        }
        if ($PSBoundParameters.ContainsKey("AdminName")) {
            $Parameters["adminName"] = $AdminName
        }
        if ($PSBoundParameters.ContainsKey("AdminPass")) {
            $Parameters["adminPass"] = $AdminPass
        }
        if ($PSBoundParameters.ContainsKey("ContactName")) {
            $Parameters["contactName"] = $ContactName
        }
        if ($PSBoundParameters.ContainsKey("Contact_phone")) {
            $Parameters["contact_phone"] = $Contact_phone
        }
        if ($PSBoundParameters.ContainsKey("Data")) {
            $Parameters["data"] = "{0}" -f ((ConvertTo-Json $Data -Compress) -replace '"', '\"')
        }
        if ($PSBoundParameters.ContainsKey("OutputFile")) {
            $Parameters["outputFile"] = $OutputFile
        }
        if ($PSBoundParameters.ContainsKey("NoProxy")) {
            $Parameters["noProxy"] = $NoProxy
        }
        if ($PSBoundParameters.ContainsKey("Session")) {
            $Parameters["session"] = $Session
        }

    }

    Process {
        foreach ($item in (PSc8y\Expand-Id $Id)) {

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (PSc8y\Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "tenants" `
                -Verb "create" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.tenant+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
