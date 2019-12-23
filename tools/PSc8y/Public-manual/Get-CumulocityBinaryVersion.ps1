Function Get-CumulocityBinaryVersion {
    [cmdletbinding()]
    Param()
    $c8y = Get-CumulocityBinary
    & $c8y version
}