. $PSScriptRoot/imports.ps1

Describe -Name "Get-ChildDeviceCollection" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $ChildDevice = PSC8y\New-TestDevice
        PSC8y\New-ChildDeviceReference -Device $Device.id -NewChild $ChildDevice.id

    }

    It "Get a list of the child devices of an existing device" {
        $Response = PSC8y\Get-ChildDeviceCollection -Device $Device.id
        $LASTEXITCODE | Should -Be 0
    }
    It "Get a list of the child devices of an existing device (using pipeline)" {
        $Response = PSC8y\Get-ManagedObject -Id $Device.id | Get-ChildDeviceCollection
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Device.id
        PSC8y\Remove-ManagedObject -Id $ChildDevice.id

    }
}

