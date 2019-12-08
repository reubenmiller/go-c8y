# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-MeasurementCollection {
<#
.SYNOPSIS
Get a collection of measurements based on filter parameters

.DESCRIPTION
Get a collection of measurements based on filter parameters

.EXAMPLE
PS> Get-MeasurementCollection
Get a list of measurements

.EXAMPLE
PS> Get-MeasurementCollection -Device $Device.id -Type "TempReading"
Get a list of measurements for a particular device

.EXAMPLE
PS> Get-DeviceCollection -Name $Device.name | Get-MeasurementCollection
Get measurements from a device (using pipeline)


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Device ID
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
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

        # Fragment name from measurement (deprecated).
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

        # Return the newest instead of the oldest measurements. Must be used with dateFrom and dateTo parameters
        [Parameter()]
        [switch]
        $Revert,

        # Results will be displayed in csv format
        [Parameter()]
        [switch]
        $Csv,

        # Results will be displayed in Excel format
        [Parameter()]
        [switch]
        $Excel,

        # Every measurement fragment which contains 'unit' property will be transformed to use required system of units.
        [Parameter()]
        [ValidateSet('imperial','metric')]
        [string]
        $Unit,

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

        # Session path
        [Parameter()]
        [string]
        $Session
    )

    Begin {
        $Parameters = @{}
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
        if ($PSBoundParameters.ContainsKey("Revert")) {
            $Parameters["revert"] = $Revert
        }
        if ($PSBoundParameters.ContainsKey("Csv")) {
            $Parameters["csv"] = $Csv
        }
        if ($PSBoundParameters.ContainsKey("Excel")) {
            $Parameters["excel"] = $Excel
        }
        if ($PSBoundParameters.ContainsKey("Unit")) {
            $Parameters["unit"] = $Unit
        }
        if ($PSBoundParameters.ContainsKey("PageSize")) {
            $Parameters["pageSize"] = $PageSize
        }
        if ($PSBoundParameters.ContainsKey("WithTotalPages") -and $WithTotalPages) {
            $Parameters["withTotalPages"] = $WithTotalPages
        }
        if ($PSBoundParameters.ContainsKey("OutputFile")) {
            $Parameters["outputFile"] = $OutputFile
        }
        if ($PSBoundParameters.ContainsKey("Session")) {
            $Parameters["session"] = $Session
        }

    }

    Process {
        $Parameters["device"] = PSC8y\Expand-Id $Device

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
            -Verb "list" `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.measurementCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.measurement+json" `
            -ResultProperty "measurements" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {}
}
