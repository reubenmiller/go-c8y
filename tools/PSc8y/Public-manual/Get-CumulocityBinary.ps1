Function Get-CumulocityBinary {
<# 
.SYNOPSIS
Get the full path to the Cumulocity Binary which is compatible with the current Operating system

.EXAMPLE
Get-CumulocityBinary

Returns the fullname of the path to the Cumulocity binary
#>
    [cmdletbinding()]
    [OutputType([String])]
    Param()

    $RootPath = "$PSScriptRoot/../Dependencies"
    if ($IsLinux) {
        Resolve-Path "$RootPath/c8y.linux"
    } elseif ($IsMacOS) {
        Resolve-Path "$RootPath/c8y.macos"
    } else {
        Resolve-Path "$RootPath/c8y.windows.exe"
    }
}
