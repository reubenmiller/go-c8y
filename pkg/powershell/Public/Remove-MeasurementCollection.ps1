# Code generated from specification version 1.0.0: DO NOT EDIT
Function Remove-MeasurementCollection {
<#
.SYNOPSIS
Delete a collection of measurements


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Device ID
        [Parameter()]
        [object[]]
        $Device,

        # Measurement type.
        [Parameter()]
        [string]
        $Type,

        # value fragment type
        [Parameter()]
        [string]
        $ValueFragmentType,

        # value fragment series
        [Parameter()]
        [string]
        $ValueFragmentSeries,

        # Fragment name from measurement.
        [Parameter()]
        [string]
        $FragmentType,

        # Start date or date and time of measurement occurrence.
        [Parameter()]
        [string]
        $DateFrom,

        # End date or date and time of measurement occurrence.
        [Parameter()]
        [string]
        $DateTo,

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
        if ($PSBoundParameters.ContainsKey("Device")) {
            $Parameters["device"] = $Device
        }
        if ($PSBoundParameters.ContainsKey("Type")) {
            $Parameters["type"] = $Type
        }
        if ($PSBoundParameters.ContainsKey("ValueFragmentType")) {
            $Parameters["valueFragmentType"] = $ValueFragmentType
        }
        if ($PSBoundParameters.ContainsKey("ValueFragmentSeries")) {
            $Parameters["valueFragmentSeries"] = $ValueFragmentSeries
        }
        if ($PSBoundParameters.ContainsKey("FragmentType")) {
            $Parameters["fragmentType"] = $FragmentType
        }
        if ($PSBoundParameters.ContainsKey("DateFrom")) {
            $Parameters["dateFrom"] = $DateFrom
        }
        if ($PSBoundParameters.ContainsKey("DateTo")) {
            $Parameters["dateTo"] = $DateTo
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
                -Noun "measurements" `
                -Verb "deleteCollection" `
                -Parameters $Parameters `
                -Type "" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
