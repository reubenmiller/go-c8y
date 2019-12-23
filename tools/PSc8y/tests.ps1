[cmdletbinding()]
Param(
    [switch] $Coverage
)
$OldConfirmPreference = $global:ConfirmPreference
$global:ConfirmPreference = "None"

if (!(Get-Module "Pester")) {
    Install-Module "Pester" -MinimumVersion "4.0.0"
    Import-Module "Pester" -MinimumVersion "4.0.0"
}

if ($Coverage) {
    $output = Invoke-Pester -Script $PSScriptRoot/Tests -CodeCoverage $PSScriptRoot/Public/* -CodeCoverageOutputFile "$PSScriptRoot/PSc8y.coverage.xml" -PassThru
} else {
    Invoke-Pester -Script $PSScriptRoot/Tests -OutputFile "report/report.xml"
}


$global:ConfirmPreference = $OldConfirmPreference
