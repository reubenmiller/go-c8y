# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-Alarm {
<#
.SYNOPSIS
Create a new alarm

.DESCRIPTION
Create a new alarm


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # The ManagedObject that the alarm originated from (required)
        [Parameter(Mandatory = $true,
                   ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Device,

        # Identifies the type of this alarm, e.g. 'com_cumulocity_events_TamperEvent'. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Type,

        # Time of the alarm. (required)
        [Parameter(Mandatory = $true)]
        [datetime]
        $Time,

        # Text description of the alarm. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Text,

        # The severity of the alarm: CRITICAL, MAJOR, MINOR or WARNING. Must be upper-case. (required)
        [Parameter(Mandatory = $true)]
        [ValidateSet('CRITICAL','MAJOR','MINOR','WARNING')]
        [string]
        $Severity,

        # The status of the alarm: ACTIVE, ACKNOWLEDGED or CLEARED. If status was not appeared, new alarm will have status ACTIVE. Must be upper-case.
        [Parameter()]
        [ValidateSet('ACTIVE','ACKNOWLEDGED','CLEARED')]
        [string]
        $Status,

        # Additional properties of the alarm.
        [Parameter()]
        [hashtable]
        $Data,

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
            -Verb new `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.alarm+json" `
            -ItemType "" `
            -ResultProperty "" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
