    It "{{ Description }}" {
        $Response = PSC8y\{{ Command }}
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }