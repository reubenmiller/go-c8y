. $PSScriptRoot/imports.ps1

Describe -Name "Get-Binary" {
    BeforeEach {
        $File = New-TestFile
        $Binary = PSC8y\New-Binary -File $File

    }

    It "Get a binary and display the contents on the console" {
        $Response = PSC8y\Get-Binary -Id $Binary.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get a binary and save it to a file" {
        $Response = PSC8y\Get-Binary -Id $Binary.id -OutputFile ./download-binary1.txt
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        PSC8y\Remove-Binary -Id $Binary.id
        if (Test-Path "./download-binary1.txt") { Remove-Item ./download-binary1.txt }
        Remove-Item $File

    }
}

