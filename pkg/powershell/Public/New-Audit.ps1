# Code generated from specification version 1.0.0: DO NOT EDIT
Function New-Audit {
<#
.SYNOPSIS
Create a new audit record


#>
    [cmdletbinding(SupportsShouldProcess = $true,
                   PositionalBinding=$true,
                   HelpUri='',
                   ConfirmImpact = 'High')]
    [Alias()]
    [OutputType([object])]
    Param(
        # Identifies the type of this audit record. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Type,

        # Time of the audit record. (required)
        [Parameter(Mandatory = $true)]
        [datetime]
        $Time,

        # Text description of the audit record. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Text,

        # An optional ManagedObject that the audit record originated from (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Source,

        # The activity that was carried out. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Activity,

        # The severity of action: critical, major, minor, warning or information. (required)
        [Parameter(Mandatory = $true)]
        [string]
        $Severity,

        # The user responsible for the audited action.
        [Parameter()]
        [string]
        $User,

        # The application used to carry out the audited action.
        [Parameter()]
        [string]
        $Application,

        # An optional collection of objects describing the changes that were carried out.
        [Parameter()]
        [object[]]
        $Changes,

        # Additional properties of the audit record.
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
            -Noun auditRecords `
            -Verb create `
            -Parameters $Parameters `
            -Type "application/vnd.com.nsn.cumulocity.auditRecord+json" `
            -ItemType "" `
            -ResultProperty "" `
            -Raw:$Raw `
            -IncludeAll:$IncludeAll
    }

    End {
        
    }
}
