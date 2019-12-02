. $PSScriptRoot/imports.ps1

Describe -Name "Get-MeasurementSeries" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $Measurement = New-TestMeasurement -Device $Device.id -Type "TempReading" -ValueFragmentType "c8y_Temperature" -ValueFragmentSeries "T"
        $Measurement2 = New-TestMeasurement -Type "TempReading" -ValueFragmentType "c8y_Temperature" -ValueFragmentSeries "T"

    }

    It "Get a list of measurements for a particular device" {
        $Response = PSC8y\Get-MeasurementSeries -Device $Device.id -Series "c8y_Temperature.T" -DateFrom "1970-01-01" -DateTo "0s"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get measurement series c8y_Temperature.T on a device" {
        $Response = PSC8y\Get-MeasurementSeries -Device $Measurement2.source.id -Series "c8y_Temperature.T" -DateFrom "1970-01-01" -DateTo "0s"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Device.id
        PSC8y\Remove-ManagedObject -Id $Measurement2.source.id

    }
}

