# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-MeasurementSeries {
<#
.SYNOPSIS
Get a collection of measurements based on filter parameters

.DESCRIPTION
Get a collection of measurements based on filter parameters

.EXAMPLE
PS> Get-MeasurementSeries -Device $Device.id -Series "c8y_Temperature.T" -DateFrom "1970-01-01" -DateTo "0s"
Get a list of measurements for a particular device

.EXAMPLE
PS> Get-MeasurementSeries -Device $Measurement2.source.id -Series "c8y_Temperature.T" -DateFrom "1970-01-01" -DateTo "0s"
Get measurement series c8y_Temperature.T on a device


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Device ID
        [Parameter()]
        [object[]]
        $Device,

        # measurement type and series name, e.g. c8y_AccelerationMeasurement.acceleration
        [Parameter()]
        [string[]]
        $Series,

        # Fragment name from measurement.
        [Parameter()]
        [string]
        $AggregationType,

        # Start date or date and time of measurement occurrence. (required)
        [Parameter(Mandatory = $true)]
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
        $Session
    )

    Begin {
        $Parameters = @{}
        if ($PSBoundParameters.ContainsKey("Device")) {
            $Parameters["device"] = $Device
        }
        if ($PSBoundParameters.ContainsKey("Series")) {
            $Parameters["series"] = $Series
        }
        if ($PSBoundParameters.ContainsKey("AggregationType")) {
            $Parameters["aggregationType"] = $AggregationType
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
                -Verb "getSeries" `
                -Parameters $Parameters `
                -Type "application/json" `
                -ItemType "" `
                -ResultProperty "" `
                -Raw:$Raw `
                -IncludeAll:$IncludeAll
        }
    }

    End {}
}
