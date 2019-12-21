. $PSScriptRoot/imports.ps1

Describe -Name "Remove-ChildAssetReference" {
    BeforeEach {
        $Group = PSC8y\New-TestDeviceGroup
        $ChildDevice = PSC8y\New-TestDevice
        PSC8y\New-ChildAssetReference -Group $Group.id -NewChildDevice $ChildDevice.id

    }

    It "Unassign a child device from its parent asset" {
        $Response = PSC8y\Remove-ChildAssetReference -Asset $Group.id -ChildDevice $ChildDevice.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $ChildDevice.id
        PSC8y\Remove-ManagedObject -Id $Group.id

    }
}

