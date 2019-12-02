. $PSScriptRoot/imports.ps1

Describe -Name "Remove-ManagedObject" {
    BeforeEach {
        $mo = PSC8y\New-ManagedObject -Name "testMO"
        $Device = PSC8y\New-TestDevice
        $ChildDevice = PSC8y\New-TestDevice
        PSC8y\New-ChildDeviceReference -Device $Device.id -NewChild $ChildDevice.id

    }

    It "Delete a managed object" {
        $Response = PSC8y\Remove-ManagedObject -Id $mo.id
        $LASTEXITCODE | Should -Be 0
    }
    It "Delete a managed object (using pipeline)" {
        $Response = PSC8y\Get-ManagedObject -Id $mo.id | Remove-ManagedObject
        $LASTEXITCODE | Should -Be 0
    }
    It "Delete a managed object and all child devices" {
        $Response = PSC8y\Get-ManagedObject -Id $Device.id | Remove-ManagedObject -Cascade
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

