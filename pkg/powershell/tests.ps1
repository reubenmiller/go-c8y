[cmdletbinding()]
Param(
    [switch] $Coverage
)
$OldConfirmPreference = $global:ConfirmPreference
$global:ConfirmPreference = "None"

if ($Coverage) {
    $output = Invoke-Pester -Script $PSScriptRoot/Tests -CodeCoverage $PSScriptRoot/Public/* -CodeCoverageOutputFile "$PSScriptRoot/PSC8y.coverage.xml" -PassThru
} else {
    Invoke-Pester -Script $PSScriptRoot/Tests
}


$global:ConfirmPreference = $OldConfirmPreference
