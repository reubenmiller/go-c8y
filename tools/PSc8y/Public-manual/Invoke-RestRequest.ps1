Function Invoke-RestRequest {
    [cmdletbinding(
        SupportsShouldProcess = $true,
        ConfirmImpact = "None")]
    Param(
        # Uri (or partial uri). i.e. /application/applications
        [Parameter(
            Mandatory = $true,
            Position = 0)]
        [string] $Uri,

        # Rest Method: defaults to GET
        [Microsoft.PowerShell.Commands.WebRequestMethod] $Method,

        [object] $Data,

        # Input file to be uploaded as FormData
        [string] $InFile,

        [hashtable] $QueryParameters
    )

    $c8y = Get-CumulocityBinary

    $c8yargs = New-Object System.Collections.ArrayList

    $null = $c8yargs.Add("rest")

    if ($Method) {
        $null = $c8yargs.Add($Method)
    }

    if ($null -ne $QueryParameters) {
        $queryparams = New-Object System.Collections.ArrayList
        foreach ($key in $QueryParameters.Keys) {
            $value = $QueryParameters[$key]
            if ($value) {
                $null = $queryparams.Add("${key}=${value}")
            }
        }

        if ($queryparams.Count -gt 0) {
            $str = $queryparams -join "&"
            if ($Uri.Contains("?")) {
                # uri already has some query parameters, so just append the new one to it
                $Uri = $Uri + "&" + $str
            } else {
                $Uri = $Uri + "?" + $str
            }
        }
    }

    $null = $c8yargs.Add($Uri)

    if ($null -ne $Data) {
        if ($Data -is [string]) {
            $null = $c8yargs.AddRange(@("--data", $Data))
        } elseif ($Data -is [hashtable]) {
            # todo: handle
            $jsonstring = ConvertTo-Json -InputObject $Data -Depth 100 -Compress
            $null = $c8yargs.AddRange(@("--data", $jsonstring))
        }
    }

    if ($InFile) {
        $null = $c8yargs.AddRange(@("--file", $InFile))
    }

    if ($WhatIfPreference) {
        $null = $c8yargs.Add("--dry")
    }

    & $c8y $c8yargs
}
