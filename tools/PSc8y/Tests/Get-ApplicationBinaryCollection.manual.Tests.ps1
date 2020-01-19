. $PSScriptRoot/imports.ps1

Describe -Name "Get-ApplicationBinaryCollection" {
    Context "existing web application" {
        $application = New-TestHostedApplication

        It "Gets a list of binaries for a given application" {
            [array] $response = Get-ApplicationBinaryCollection -Id $application.id

            $LASTEXITCODE | Should -Be 0
            $application | Should -Not -BeNullOrEmpty
            $response | Should -HaveCount 1

            $application.activeVersionId | Should -BeExactly $response[0].id
        }

        PSc8y\Remove-Application -Id $application.id
    }
}
