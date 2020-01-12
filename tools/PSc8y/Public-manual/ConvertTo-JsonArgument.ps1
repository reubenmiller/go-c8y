Function ConvertTo-JsonArgument {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true,
            Position = 0
        )]
        [object] $Data
    )

    if ($Data -is [string]) {
        # If string, then validate if json was provided
        $DataObj = (ConvertFrom-Json $Data)
    } else {
        $DataObj = $Data
    }

    $strArg = "{0}" -f ((ConvertTo-Json $DataObj -Compress) -replace '"', '\"')

    # Replace space with unicode char, as space can have console parsing problems
    $strArg = $strArg -replace " ", "\u0020"
    $strArg
}
