. $PSScriptRoot/imports.ps1

Describe -Name "Get-TenantUsageSummaryStatistics" {
    BeforeEach {

    }

    It "Get tenant summary statistics for the current tenant" {
        $Response = PSC8y\Get-TenantUsageSummaryStatistics
        $LASTEXITCODE | Should -Be 0
    }
    It "Get tenant summary statistics collection for the last 30 days" {
        $Response = PSC8y\Get-TenantUsageSummaryStatistics -DateFrom "-30d"
        $LASTEXITCODE | Should -Be 0
    }
    It "Get tenant summary statistics collection for the last 10 days, only return until the last 9 days" {
        $Response = PSC8y\Get-TenantUsageSummaryStatistics -DateFrom "-10d" -DateTo "-9d"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

