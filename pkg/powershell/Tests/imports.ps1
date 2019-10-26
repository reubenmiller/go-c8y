
Remove-Module PSC8y -ErrorAction SilentlyContinue

Write-Verbose "PSScriptRoot: $PSSScriptRoot";
Import-Module Pester -MinimumVersion "4.0.0"
Import-Module $PSScriptRoot/../PSC8y.psd1 -Prefix ""

# Get credentials from the environment
$Session = Get-C8yActiveSession -ErrorAction SilentlyContinue

if (!$Session.id) {
    New-C8ySessionFromEnvironment;
}
Write-Host ("Session: {0}/{1} on {2}" -f $Session.Tenant, $Session.Username, $Session.Uri)
