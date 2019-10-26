# Code generated from specification version 1.0.0: DO NOT EDIT
Function Update-Operation {
<#
.SYNOPSIS
Update operation

.DESCRIPTION
Update operation


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Operation id
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [string]
        $Id,

        # Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING. (required)
        [Parameter(Mandatory = $true)]
        [ValidateSet('PENDING','EXECUTING','SUCCESSFUL','FAILED')]
        [string]
        $Status,

        # Reason for the failure. Use whne setting status to FAILED
        [Parameter()]
        [string]
        $FailureReason,

        # Additional properties describing the operation which will be performed on the device.
        [Parameter()]
        [hashtable]
        $Data,

        # Include raw response including pagination information
        [Parameter()]
        [switch]
        $Raw
    )

    Begin {
        $Parameters = @{}
        if ($PSBoundParameters.ContainsKey("Id")) {
            $Parameters["id"] = $Id
        }
        if ($PSBoundParameters.ContainsKey("Status")) {
            $Parameters["status"] = $Status
        }
        if ($PSBoundParameters.ContainsKey("FailureReason")) {
            $Parameters["failureReason"] = $FailureReason
        }
        if ($PSBoundParameters.ContainsKey("Data")) {
            $Parameters["data"] = "{0}" -f ((ConvertTo-Json $Data -Compress) -replace '"', '\"')
        }

    }

    Process {
        foreach ($item in @($Id)) {

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "operations" `
                -Verb "update" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.operation+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
