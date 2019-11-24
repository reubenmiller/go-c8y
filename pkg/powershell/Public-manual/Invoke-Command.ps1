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

    $args = New-Object System.Collections.ArrayList
    $null = $args.Add($Noun)
    $null = $args.Add($Verb)

    foreach ($iKey in $Parameters.Keys) {
        $Value = $Parameters[$iKey]

        foreach ($iValue in $Value) {
            if ("$Value" -notmatch "^$") {
                $key = $iKey[0].ToString().ToLowerInvariant() + $iKey.SubString(1)
                if ($Value -is [bool] -and $Value) {
                    $null = $args.AddRange(@("--${key}"))
                } else {
                    if ($key -eq "data") {
                        # due to cli parsing, data needs to be sent using "="
                        $null = $args.AddRange(@("--${key}", $Value))
                    } else {
                        $null = $args.Add("--${key}=$Value")
                    }
                }
            }
        }
    }

    $null = $args.Add("--pretty=false")

    if ($WhatIfPreference) {
        $null = $args.Add("--dry")
    }

    if ($VerbosePreference) {
        $null = $args.Add("--verbose")
    }

    # Include all pagination results
    if ($IncludeAll) {
        $null = $args.Add("--all")
    }

    $null = $args.Add("--raw")

    $c8ycli = Get-CumulocityBinary
    Write-Verbose ("$c8ycli {0}" -f $args -join " ")

    $RawResponse = & $c8ycli $args

    $ExitCode = $LASTEXITCODE
    if ($ExitCode -ne 0) {

        try {
            $errormessage = $RawResponse | Select-Object -First 1 | ConvertFrom-Json
            Write-Error ("{0}: {1}" -f @(
                $errormessage.error,
                $errormessage.message
            ))
        } catch {
            Write-Error "c8y command failed. $RawResponse"
        }
        return
    }

    $response = $RawResponse | ConvertFrom-Json

    $NestedData = Get-NestedProperty -InputObject $response -Name $ResultProperty

    if ($ResultProperty -and $ItemType) {
        $null = $NestedData `
            | Select-Object `
            | Add-PowershellType $ItemType
    }

    if ($response -and $Type) {
        $null = $response `
            | Select-Object `
            | Add-PowershellType $Type
    }

    $ReturnRawData = $Raw -or [string]::IsNullOrEmpty($ResultProperty) -or (
        $Parameters.ContainsKey("WithTotalPages") -and
        $Parameters["WithTotalPages"]
    )

    if ($response.statistics.pageSize) {
        Write-Verbose ("Statistics: currentPage={2}, pageSize={0}, totalPages={1}" -f @(
            $response.statistics.pageSize,
            $response.statistics.totalPages,
            $response.statistics.currentPage
        ))
    }

    if ($NestedData) {
        $null = Add-Member -InputObject $NestedData -MemberType NoteProperty -Name "PSStatistics" -Value @{
            pageSize = $response.statistics.pageSize
            totalPages = $response.statistics.totalPages
            currentPage = $response.statistics.currentPage
        }
    }

    # Save last value for easier recall on command line
    $global:_rawdata = $response
    $global:_data = $null


    if ($null -ne $NestedData -or $NestedData.Count -ge 0) {
        $global:_data = $NestedData
    }

    if ($ReturnRawData -or
        ($null -eq $NestedData -and $null -eq $NestedData.Count)) {
        $response
    } else {
        $NestedData
    }
}
