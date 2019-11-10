Function Get-NestedProperty {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true,
            Position = 0
        )]
        [AllowNull()]
        [object[]] $InputObject,

        [Parameter(
            Mandatory = $true,
            Position = 1
        )]
        [AllowNull()]
        [AllowEmptyString()]
        [string] $Name
    )

    if (!$Name) {
        $null
        return
    }

    $Output = $InputObject

    foreach ($part in ($Name -split "\.")) {
        if ($null -eq $Output.$part) {
            $Output = $null
            break;
        }
        $Output = $Output.$part
    }
    $Output
}