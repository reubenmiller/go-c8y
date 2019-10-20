# Code generated from specification version 1.0.0: DO NOT EDIT
Function Get-OperationCollection {
<#
.SYNOPSIS
Get a collection of operations based on filter parameters

.DESCRIPTION
Get a collection of operations based on filter parameters

.EXAMPLE
Get a list of pending operations
Get-OperationCollection -Status PENDING


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'None')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Agent ID
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Agent,

        # Device ID
        [Parameter(ValueFromPipeline=$true,
                   ValueFromPipelineByPropertyName=$true)]
        [object[]]
        $Device,

        # Start date or date and time of operation.
        [Parameter()]
        [datetime]
        $DateFrom,

        # End date or date and time of operation.
        [Parameter()]
        [datetime]
        $DateTo,

        # Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING.
        [Parameter()]
        [ValidateSet('PENDING','EXECUTING','SUCCESSFUL','FAILED')]
        [string]
        $Status,

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
            -Noun operations `
            -Verb list `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.operationCollection+json" `
            -ItemType "application/vnd.com.nsn.cumulocity.operation+json" `
            -ResultProperty "operations" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
