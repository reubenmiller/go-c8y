Function Get-CumulocityBinary {
    $RootPath = "$PSScriptRoot/../Dependencies"
    if ($IsLinux) {
        Resolve-Path "$RootPath/c8y.linux"
    } elseif ($IsMacOS) {
        Resolve-Path "$RootPath/c8y.macos"
    } else {
        Resolve-Path "$RootPath/c8y.windows.exe"
    }
}
