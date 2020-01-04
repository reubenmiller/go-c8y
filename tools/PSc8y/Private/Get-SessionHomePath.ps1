Function Get-SessionHomePath {
    [cmdletbinding()]
    Param()

    if ($env:C8Y_SESSION_HOME) {
        $HomePath = $env:C8Y_SESSION_HOME
    }
    elseif ($env:HOME) {
        $HomePath = Join-Path $env:HOME -ChildPath ".cumulocity"
    }
    else {
        # default to current directory
        $HomePath = Join-Path "." -ChildPath ".cumulocity"
    }

    $HomePath
}
