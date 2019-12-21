Function Get-CumulocityBinary {
    if ($IsLinux -or $IsMacOS) {
        Resolve-Path "$PSScriptRoot/../c8y"
    } else {
        Resolve-Path "$PSScriptRoot/../c8y.exe"
    }
}