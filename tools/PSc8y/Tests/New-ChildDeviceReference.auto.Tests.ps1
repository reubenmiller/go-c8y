. $PSScriptRoot/imports.ps1

Describe -Name "New-ChildDeviceReference" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $ChildDevice = PSC8y\New-TestDevice

    }

    It "Assign a device as a child device to an existing device" {
        $Response = PSC8y\New-ChildDeviceReference -Device $Device.id -NewChild $ChildDevice.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Assign a device as a child device to an existing device (using pipeline)" {
        $Response = PSC8y\Get-ManagedObject -Id $ChildDevice.id | New-ChildDeviceReference -Device $Device.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $ChildDevice.id
        PSC8y\Remove-ManagedObject -Id $Device.id

    }
}

