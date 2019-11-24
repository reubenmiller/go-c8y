Function Expand-Id {
<#
.SYNOPSIS
Expand a list of ids.

.PARAMETER InputObject
List of ids

.EXAMPLE
Expand-Id 12345

Normalize a list of ids

.EXAMPLE
"12345", "56789" | Expand-Id

Normalize a list of ids

#>
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory=$true,
            ValueFromPipeline=$true,
            Position=0
        )]
        [AllowEmptyCollection()]
        [AllowNull()]
        [object[]] $InputObject
    )

    Process {
        [array] $AllIds = foreach ($iID in $InputObject)
        {
            if (($iID -is [string]) -and ($iID -match "^\d+$"))
            {
                $iID

            }
        }
        $AllIds
    }
}
