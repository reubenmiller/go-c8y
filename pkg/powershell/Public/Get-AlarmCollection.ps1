# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-AlarmCollection {
<#
.SYNOPSIS
Get a collection of alarms based on filter parameters

.DESCRIPTION
Get a collection of alarms based on filter parameters

.EXAMPLE
Get-AlarmCollection


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Source device id.
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Device,

        # Start date or date and time of alarm occurrence.
        [Parameter()]
        [datetime]
        $DateFrom,

        # End date or date and time of alarm occurrence.
        [Parameter()]
        [datetime]
        $DateTo,

        # Alarm type.
        [Parameter()]
        [string]
        $Type,

        # Alarm fragment type.
        [Parameter()]
        [string]
        $FragmentType,

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

        # When set to true only resolved alarms will be removed (the one with status CLEARED), false means alarms with status ACTIVE or ACKNOWLEDGED.
        [Parameter()]
        [switch]
        $Resolved,

        # Include assets
        [Parameter()]
        [switch]
        $WithAssets,

        # Include devices
        [Parameter()]
        [switch]
        $WithDevices,

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
            } elseif ($iParam.Value -is [hashtable]) {
                $Parameters[$Name] = "{0}" -f ((ConvertTo-Json $iParam.Value -Compress) -replace '"', '\"')
            } elseif ($iParam.Value -is [datetime]) {
                $Parameters[$Name] = Format-Date $iParam.Value
            } else {
                if ("$iParam" -notmatch "^$") {
                    $Parameters[$Name] = $iParam.Value
                }
            }
        }

        Invoke-Command `
            -Noun alarms `
            -Verb list `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.alarmCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.alarm+json" `
            -ResultProperty "alarms" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
