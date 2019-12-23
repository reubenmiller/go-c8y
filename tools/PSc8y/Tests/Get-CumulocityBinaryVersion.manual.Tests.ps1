. $PSScriptRoot/imports.ps1

Describe -Name "Get-CumulocityBinaryVersion" {

    It "should show a version number" {
        $Response = Get-CumulocityBinaryVersion | Select-Object -Last 1
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Match "\d+\.\d+\.\d+"
    }

}
