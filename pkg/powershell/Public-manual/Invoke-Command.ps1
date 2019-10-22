Function Invoke-Command {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true
        )]
        [string] $Noun,

        [Parameter(
            Mandatory = $true
        )]
        [string] $Verb,

        [hashtable] $Parameters,

        [string] $Type = "c8y.item",

        [string] $ItemType,

        [string] $ResultProperty,

        [switch] $IncludeAll,

        [switch] $Raw
    )

    $BinaryArguments = New-Object System.Collections.ArrayList
    $null = $BinaryArguments.Add($Noun)
    $null = $BinaryArguments.Add($Verb)

    foreach ($iKey in $Parameters.Keys) {
        $Value = $Parameters[$iKey]

        foreach ($iValue in $Value) {
            if ("$Value" -notmatch "^$") {
                $key = $iKey[0].ToString().ToLowerInvariant() + $iKey.SubString(1)
                if ($Value -is [bool] -and $Value) {
                    $null = $BinaryArguments.AddRange(@("--${key}"))
                } else {
                    $null = $BinaryArguments.AddRange(@("--${key}=$Value"))
                }
            }
        }
    }

    $null = $BinaryArguments.Add("--pretty=false")

    # Include all pagination results
    if ($IncludeAll) {
        $null = $BinaryArguments.Add("--all")
    }

    Write-Verbose ("./c8y.exe {0}" -f $BinaryArguments -join " ")

    $response = & ./c8y.exe $BinaryArguments | ConvertFrom-Json

    if ($ResultProperty -and $ItemType) {
        $null = $response.$ResultProperty | Add-PowershellType $ItemType
    }

    if ($response -and $Type) {
        $null = $response | Add-PowershellType $Type
    }

    $ReturnRawData = $Raw -or [string]::IsNullOrEmpty($ResultProperty) -or (
        $Parameters.ContainsKey("WithTotalPages") -and
        $Parameters["WithTotalPages"]
    )

    Write-Verbose ("Statistics: currentPage={2}, pageSize={0}, totalPages={1}" -f @(
        $response.statistics.pageSize,
        $response.statistics.totalPages,
        $response.statistics.currentPage
    ))

    if ($response.$ResultProperty) {
        $null = Add-Member -InputObject $response.$ResultProperty -MemberType NoteProperty -Name "PSStatistics" -Value @{
            pageSize = $response.statistics.pageSize
            totalPages = $response.statistics.totalPages
            currentPage = $response.statistics.currentPage
        }
    }

    if ($ReturnRawData -or ($null -eq $response.$ResultProperty)) {
        $response
    } else {
        $response.$ResultProperty
    }
}
