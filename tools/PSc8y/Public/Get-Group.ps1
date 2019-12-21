# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-Group {
<#
.SYNOPSIS
Create a new group by id

.EXAMPLE
PS> Get-Group -Id $Group.id
Get a user group


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

        # Group id
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [string]
        $Id,

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
        $Session
    )

    Begin {
        $Parameters = @{}
        if ($PSBoundParameters.ContainsKey("Tenant")) {
            $Parameters["tenant"] = $Tenant
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
        $Parameters["id"] = PSC8y\Expand-Id $Id

        if (!$Force -and
            !$WhatIfPreference -and
            !$PSCmdlet.ShouldProcess(
                (PSC8y\Get-C8ySessionProperty -Name "tenant"),
                (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
            )) {
            continue
        }

        Invoke-Command `
            -Noun "userGroups" `
            -Verb "get" `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.group+json" `
            -ItemType "" `
            -ResultProperty "" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {}
}
