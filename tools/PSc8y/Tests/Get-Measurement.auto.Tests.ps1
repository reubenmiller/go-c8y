. $PSScriptRoot/imports.ps1

Describe -Name "Get-Measurement" {
    BeforeEach {
        $Measurement = New-TestMeasurement

    }

    It "Get measurement" {
        $Response = PSC8y\Get-Measurement -Id $Measurement.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Measurement.source.id

    }
}

