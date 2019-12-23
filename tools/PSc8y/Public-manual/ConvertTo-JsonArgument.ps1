Function ConvertTo-JsonArgument {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true,
            Position = 0
        )]
        [object] $Data
    )
    $strArg = "{0}" -f ((ConvertTo-Json $Data -Compress) -replace '"', '\"')

    # Replace space with unicode char, as space can have console parsing problems
    $strArg = $strArg -replace " ", "\u0020"
    $strArg
}
