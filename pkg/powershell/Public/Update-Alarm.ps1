# Code generated from specification version 1.0.0: DO NOT EDIT
Function Update-Alarm {
<#
.SYNOPSIS
Update an alarm

.EXAMPLE
PS> Update-Alarm -Id $Alarm.id -Status ACKNOWLEDGED
Acknowledge an existing alarm

.EXAMPLE
PS> Update-Alarm -Id $Alarm.id -Severity CRITICAL
Update severity of an existing alarm to CRITICAL


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Alarm id (required)
        [Parameter(Mandatory = $true,
                   ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [string]
        $Id,

        # Comma separated alarm statuses, for example ACTIVE,CLEARED.
        [Parameter()]
        [ValidateSet('ACTIVE','ACKNOWLEDGED','CLEARED')]
        [string]
        $Status,

        # Alarm severity, for example CRITICAL, MAJOR, MINOR or WARNING.
        [Parameter()]
        [ValidateSet('CRITICAL','MAJOR','MINOR','WARNING')]
        [string]
        $Severity,

        # Text description of the alarm.
        [Parameter()]
        [string]
        $Text,

        # Additional properties of the alarm.
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
        if ($PSBoundParameters.ContainsKey("Status")) {
            $Parameters["status"] = $Status
        }
        if ($PSBoundParameters.ContainsKey("Severity")) {
            $Parameters["severity"] = $Severity
        }
        if ($PSBoundParameters.ContainsKey("Text")) {
            $Parameters["text"] = $Text
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
        foreach ($item in (PSC8y\Expand-Id $Id)) {
            if ($item) {
                $Parameters["id"] = if ($item.id) { $item.id } else { $item }
            }

            if (!$Force -and
                !$WhatIfPreference -and
                !$PSCmdlet.ShouldProcess(
                    (PSC8y\Get-C8ySessionProperty -Name "tenant"),
                    (Format-ConfirmationMessage -Name $PSCmdlet.MyInvocation.InvocationName -InputObject $item)
                )) {
                continue
            }

            Invoke-Command `
                -Noun "alarms" `
                -Verb "update" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.alarm+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
