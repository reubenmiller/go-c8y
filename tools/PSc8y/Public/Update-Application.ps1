# Code generated from specification version 1.0.0: DO NOT EDIT
Function Update-Application {
<#
.SYNOPSIS
Update application meta information

.EXAMPLE
PS> Update-Application -Application "helloworld-app" -Availability "MARKET"
Update application availability to MARKET


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Application id (required)
        [Parameter(Mandatory = $true,
                   ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Application,

        # data
        [Parameter()]
        [hashtable]
        $Data,

        # Name of application
        [Parameter()]
        [string]
        $Name,

        # Shared secret of application
        [Parameter()]
        [string]
        $Key,

        # Access level for other tenants.  Possible values are : MARKET, PRIVATE (default)
        [Parameter()]
        [ValidateSet('MARKET','PRIVATE')]
        [string]
        $Availability,

        # contextPath of the hosted application
        [Parameter()]
        [string]
        $ContextPath,

        # URL to application base directory hosted on an external server
        [Parameter()]
        [string]
        $ResourcesUrl,

        # authorization username to access resourcesUrl
        [Parameter()]
        [string]
        $ResourcesUsername,

        # authorization password to access resourcesUrl
        [Parameter()]
        [string]
        $ResourcesPassword,

        # URL to the external application
        [Parameter()]
        [string]
        $ExternalUrl,

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
        if ($PSBoundParameters.ContainsKey("Data")) {
            $Parameters["data"] = ConvertTo-JsonArgument $Data
        }
        if ($PSBoundParameters.ContainsKey("Name")) {
            $Parameters["name"] = $Name
        }
        if ($PSBoundParameters.ContainsKey("Key")) {
            $Parameters["key"] = $Key
        }
        if ($PSBoundParameters.ContainsKey("Availability")) {
            $Parameters["availability"] = $Availability
        }
        if ($PSBoundParameters.ContainsKey("ContextPath")) {
            $Parameters["contextPath"] = $ContextPath
        }
        if ($PSBoundParameters.ContainsKey("ResourcesUrl")) {
            $Parameters["resourcesUrl"] = $ResourcesUrl
        }
        if ($PSBoundParameters.ContainsKey("ResourcesUsername")) {
            $Parameters["resourcesUsername"] = $ResourcesUsername
        }
        if ($PSBoundParameters.ContainsKey("ResourcesPassword")) {
            $Parameters["resourcesPassword"] = $ResourcesPassword
        }
        if ($PSBoundParameters.ContainsKey("ExternalUrl")) {
            $Parameters["externalUrl"] = $ExternalUrl
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
        foreach ($item in (PSc8y\Expand-Application $Application)) {
            if ($item) {
                $Parameters["application"] = if ($item.id) { $item.id } else { $item }
            }

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (PSc8y\Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "applications" `
                -Verb "update" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.application+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
