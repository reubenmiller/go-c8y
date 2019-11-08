Function Get-Session {
<#
.SYNOPSIS
Get the active Cumulocity Session

.EXAMPLE
Get-Session

Get the current Cumulocity session

.OUTPUTS
None
#>
    [CmdletBinding()]
    Param()
    $Path = $env:C8Y_SESSION

    if (!$Path) {
        Write-Warning "No active session is set"
        return
    }

    if (!(Test-Path $Path)) {
        Write-Error "Session file does not exist"
        return
    }

    $data = Get-Content -LiteralPath $Path | ConvertFrom-Json
    $data | Add-Member -MemberType NoteProperty -Name "path" -Value $Path -ErrorAction SilentlyContinue
    $data.path = (Resolve-Path $Path).ProviderPath
    $data | Add-PowershellType "cumulocity/session"
}
