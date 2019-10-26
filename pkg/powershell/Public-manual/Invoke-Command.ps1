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
                    if ($key -eq "data") {
                        # due to cli parsing, data needs to be sent using "="
                        $null = $BinaryArguments.AddRange(@("--${key}", $Value))
                    } else {
                        $null = $BinaryArguments.Add("--${key}=$Value")
                    }
                }
            }
        }
    }

    $null = $BinaryArguments.Add("--pretty=false")

    if ($WhatIfPreference) {
        $null = $BinaryArguments.Add("--dry")
    }

    # Include all pagination results
    if ($IncludeAll) {
        $null = $BinaryArguments.Add("--all")
    }

    $Binary = Resolve-Path "$PSScriptRoot/../c8y.exe"
    Write-Verbose ("$Binary {0}" -f $BinaryArguments -join " ")

    $RawResponse = & $Binary $BinaryArguments

    $ExitCode = $LASTEXITCODE
    if ($ExitCode -ne 0) {

        try {
            $errormessage = $RawResponse | Select-Object -First 1 | ConvertFrom-Json
            Write-Error ("{0}: {1}" -f @(
                $errormessage.error,
                $errormessage.message
            ))
        } catch {
            Write-Error "c8y command failed for an unknown reason. $RawResponse"
        }
        return
    }

    $response = $RawResponse | ConvertFrom-Json

    if ($ResultProperty -and $ItemType) {
        $null = $response.$ResultProperty `
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
