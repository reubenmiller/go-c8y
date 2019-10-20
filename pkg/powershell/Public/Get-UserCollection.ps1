# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-UserCollection {
<#
.SYNOPSIS
Get a collection of users based on filter parameters

.DESCRIPTION
Get a collection of users based on filter parameters

.EXAMPLE
Get a list of users
Get-UserCollection


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Tenant
        [Parameter()]
        [string]
        $Tenant,

        # prefix or full username
        [Parameter()]
        [string]
        $Username,

        # numeric group identifiers separated by commas; result will contain only users which belong to at least one of specified groups
        [Parameter()]
        [string]
        $Groups,

        # exact username
        [Parameter()]
        [string]
        $Owner,

        # If set to 'true', result will contain only users created during bootstrap process (starting with 'device_'). If flag is absent (or false) the result will not contain 'device_' users.
        [Parameter()]
        [switch]
        $OnlyDevices,

        # if set to 'true', then each of returned users will contain additional field 'subusersCount' - number of direct subusers (users with corresponding 'owner').
        [Parameter()]
        [switch]
        $WithSubusersCount,

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
            -Noun users `
            -Verb list `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.userCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.user+json" `
            -ResultProperty "users" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
