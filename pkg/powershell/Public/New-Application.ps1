# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-Application {
<#
.SYNOPSIS
New application


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # data
        [Parameter()]
        [hashtable]
        $Data,

        # Name of application
        [Parameter()]
        [string]
        $Name,

        # Shared secret of application
        [Parameter()]
        [string]
        $Key,

        # Type of application. Possible values are EXTERNAL, HOSTED, MICROSERVICE
        [Parameter()]
        [ValidateSet('EXTERNAL','HOSTED','MICROSERVICE')]
        [string]
        $Type,

        # Application will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *].
        [Parameter()]
        [ValidateSet('MARKET','PRIVATE')]
        [string]
        $Availability,

        # contextPath of the hosted application. Required when application type is HOSTED
        [Parameter()]
        [string]
        $ContextPath,

        # URL to application base directory hosted on an external server. Required when application type is HOSTED
        [Parameter()]
        [string]
        $ResourcesUrl,

        # authorization username to access resourcesUrl
        [Parameter()]
        [string]
        $ResourcesUsername,

        # authorization password to access resourcesUrl
        [Parameter()]
        [string]
        $ResourcesPassword,

        # URL to the external application. Required when application type is EXTERNAL
        [Parameter()]
        [string]
        $ExternalUrl,

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
            -Noun applications `
            -Verb create `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.application+json" `
            -ItemType "" `
            -ResultProperty "" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
