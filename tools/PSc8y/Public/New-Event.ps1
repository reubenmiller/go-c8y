# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-Event {
<#
.SYNOPSIS
Create event

.DESCRIPTION
Create event

.EXAMPLE
PS> New-Event -Device "{{ randomdevice }}" -Type c8y_TestAlarm -Time "-0s" -Text "Test event"
Create a new event for a device


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # The ManagedObject which is the source of this event. (required)
        [Parameter(Mandatory = $true)]
        [object[]]
        $Device,

        # Time of the event. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Time,

        # Identifies the type of this event. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Type,

        # Text description of the event. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Text,

        # Additional properties of the event.
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
        if ($PSBoundParameters.ContainsKey("Device")) {
            $Parameters["device"] = $Device
        }
        if ($PSBoundParameters.ContainsKey("Time")) {
            $Parameters["time"] = $Time
        }
        if ($PSBoundParameters.ContainsKey("Type")) {
            $Parameters["type"] = $Type
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
                -Noun "events" `
                -Verb "create" `
                -Parameters $Parameters `
                -Type "application/vnd.com.nsn.cumulocity.event+json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
