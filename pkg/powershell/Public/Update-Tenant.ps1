# Code generated from specification version 1.0.0: DO NOT EDIT
Function Update-Tenant {
<#
.SYNOPSIS
Update tenant


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Tenant id (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Id,

        # Company name. Maximum 256 characters (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Company,

        # Domain name to be used for the tenant. Maximum 256 characters (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Domain,

        # Username of the tenant administrator
        [Parameter()]
        [string]
        $AdminName,

        # Password of the tenant administrator
        [Parameter()]
        [string]
        $AdminPass,

        # A contact name, for example an administrator, of the tenant
        [Parameter()]
        [string]
        $ContactName,

        # An international contact phone number
        [Parameter()]
        [string]
        $Contact_phone,

        # A set of custom properties of the tenant
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
            } elseif ($iParam.Value -is [datetime]) {
                $Parameters[$Name] = Format-Date $iParam.Value
            } else {
                if ("$iParam" -notmatch "^$") {
                    $Parameters[$Name] = $iParam.Value
                }
            }
        }

        Invoke-Command `
            -Noun tenants `
            -Verb update `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.tenant+json" `
            -ItemType "" `
            -ResultProperty "" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
