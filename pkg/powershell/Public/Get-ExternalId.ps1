# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-ExternalId {
<#
.SYNOPSIS
Get external id


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # External identity type (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Type,

        # External identity id/name (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Name,

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
            -Noun identity `
            -Verb get `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.externalId+json" `
            -ItemType "" `
            -ResultProperty "" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
