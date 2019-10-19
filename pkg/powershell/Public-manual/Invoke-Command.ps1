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

        [hashtable] $Parameters
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

    Write-Verbose ("./c8y.exe {0}" -f $BinaryArguments -join " ")

    & ./c8y.exe $BinaryArguments
}
