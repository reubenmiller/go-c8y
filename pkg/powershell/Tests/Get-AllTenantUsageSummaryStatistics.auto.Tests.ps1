. $PSScriptRoot/imports.ps1

Describe -Name "Get-AllTenantUsageSummaryStatistics" {
    BeforeEach {

    }

    It "Get tenant summary statistics for all tenants" {
        $Response = PSC8y\Get-AllTenantUsageSummaryStatistics
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get tenant summary statistics collection for the last 30 days" {
        $Response = PSC8y\Get-AllTenantUsageSummaryStatistics -DateFrom "-30d" -PageSize 30
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get tenant summary statistics collection for the last 10 days, only return until the last 9 days" {
        $Response = PSC8y\Get-AllTenantUsageSummaryStatistics -DateFrom "-10d" -DateTo "-9d"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {

    }
}

