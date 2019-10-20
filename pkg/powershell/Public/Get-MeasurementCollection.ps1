# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-MeasurementCollection {
<#
.SYNOPSIS
Get a collection of measurements based on filter parameters

.DESCRIPTION
Get a collection of measurements based on filter parameters

.EXAMPLE
Get a list of measurements
Get-MeasurementCollection


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

        # Fragment name from measurement.
        [Parameter()]
        [string]
        $FragmentType,

        # Start date or date and time of measurement occurrence.
        [Parameter()]
        [datetime]
        $DateFrom,

        # End date or date and time of measurement occurrence.
        [Parameter()]
        [datetime]
        $DateTo,

        # Return the newest instead of the oldest measurements. Must be used with dateFrom and dateTo parameters
        [Parameter()]
        [switch]
        $Revert,

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
        $Raw
    )

    Begin {
        
    }

    Process {
        # Get the command name
        $CommandName = $PSCmdlet.MyInvocation.InvocationName;
        # Get the list of parameters for the command
        $ParameterList = (Get-Command -Name $CommandName).Parameters;

        $Parameters = @{}

        # Grab each parameter value, using Get-Variable
        foreach ($Name in ($ParameterList.Keys -notmatch "^Raw$")) {
            $iParam = Get-Variable -Name $Name -ErrorAction SilentlyContinue;

            if ($iParam.Value -is [Switch]) {
                if ($iParam.Value.IsPresent -and $iParam) {
                    $Parameters[$Name] = $true
                }
            } elseif ($iParam.Value -is [datetime]) {
                $Parameters[$Name] = Format-Date $iParam.Value
            } else {
                if ("$iParam" -notmatch "^$") {
                    $Parameters[$Name] = $iParam.Value
                }
            }
        }

        Invoke-Command `
            -Noun measurements `
            -Verb list `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.measurementCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.measurement+json" `
            -ResultProperty "measurements" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
