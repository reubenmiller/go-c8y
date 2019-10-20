# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-RoleReferenceCollectionFromUser {
<#
.SYNOPSIS
Get collection of user role references from a user


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

        # Username (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Username,

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
            -Noun userRoles `
            -Verb getRoleReferenceCollectionFromUser `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.roleReferenceCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.roleReference+json" `
            -ResultProperty "references" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
