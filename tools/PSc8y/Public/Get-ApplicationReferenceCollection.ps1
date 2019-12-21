# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-ApplicationReferenceCollection {
<#
.SYNOPSIS
Enable application on tenant


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Tenant id
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object]
        $Tenant,

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
        if ($PSBoundParameters.ContainsKey("PageSize")) {
            $Parameters["pageSize"] = $PageSize
        }
        if ($PSBoundParameters.ContainsKey("WithTotalPages") -and $WithTotalPages) {
            $Parameters["withTotalPages"] = $WithTotalPages
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
        $Parameters["tenant"] = PSC8y\Expand-Id $Tenant

        if (!$Force -and
            !$WhatIfPreference -and
            !$PSCmdlet.ShouldProcess(
                (PSC8y\Get-C8ySessionProperty -Name "tenant"),
                (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
            )) {
            continue
        }

        Invoke-Command `
            -Noun "tenants" `
            -Verb "listReferences" `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.applicationReferenceCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.applicationReference+json" `
            -ResultProperty "references" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {}
}
