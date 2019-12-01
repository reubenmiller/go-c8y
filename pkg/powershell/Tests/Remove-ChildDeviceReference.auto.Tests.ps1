. $PSScriptRoot/imports.ps1

Describe -Name "Remove-ChildDeviceReference" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $ChildDevice = PSC8y\New-TestDevice
        PSC8y\New-ChildDeviceReference -Device $Device.id -NewChild $ChildDevice.id

    }

    It "Unassign a child device from its parent device" {
        $Response = PSC8y\Remove-ChildDeviceReference -Device $Device.id -ChildDevice $ChildDevice.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $ChildDevice.id
        PSC8y\Remove-ManagedObject -Id $Device.id

    }
}

