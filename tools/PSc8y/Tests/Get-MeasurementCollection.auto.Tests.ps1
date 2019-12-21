. $PSScriptRoot/imports.ps1

Describe -Name "Get-MeasurementCollection" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $Measurement = New-TestMeasurement -Device $Device.id -Type "TempReading"

    }

    It "Get a list of measurements" {
        $Response = PSC8y\Get-MeasurementCollection
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get a list of measurements for a particular device" {
        $Response = PSC8y\Get-MeasurementCollection -Device $Device.id -Type "TempReading"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get measurements from a device (using pipeline)" {
        $Response = PSC8y\Get-DeviceCollection -Name $Device.name | Get-MeasurementCollection
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Device.id

    }
}

