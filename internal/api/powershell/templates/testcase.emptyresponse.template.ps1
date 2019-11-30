    It "{{ Description }}" {
        $Response = PSC8y\{{ Command }}
        $LASTEXITCODE | Should -Be 0
    }