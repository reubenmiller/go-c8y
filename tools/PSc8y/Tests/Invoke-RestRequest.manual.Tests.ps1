. $PSScriptRoot/imports.ps1

Describe -Name "Invoke-RestRequest" {

    It "gets a list of applications (defaults to GET method)" {
        $Response = Invoke-RestRequest -Uri "/application/applications"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    It "should accept query parameters" {
        $Response = Invoke-RestRequest -Uri "/alarm/alarms" -QueryParameters @{
            pageSize = "1";
        } -Whatif 2>&1
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
        ($Response -join "`n") | Should -BeLike "*/alarm/alarms?pageSize=1*"
    }

    It "post an inventory managed object from a string" {
        $Response = Invoke-RestRequest -Uri "/inventory/managedObjects" -Method "post" -Data "name=test"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty

        $obj = $Response | ConvertFrom-Json
        $obj.name | Should -BeExactly "test"

        if ($obj.id) {
            Remove-ManagedObject -Id $obj.id
        }
    }

    It "Uploads a file to the inventory api" {
        $Text = "äüöp01!"
        $TestFile = New-TestFile -InputObject $Text
        $Response = Invoke-RestRequest -Uri "inventory/binaries" -Method "post" -InFile $TestFile
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty

        $obj = $Response | ConvertFrom-Json
        $obj.name | Should -BeExactly (Get-Item $TestFile).Name

        # Download file
        $BinaryContents = Get-Binary -Id $obj.id
        $BinaryContents | Should -BeExactly $Text

        # Cleanup
        Remove-Item $TestFile

        if ($obj.id) {
            Remove-Binary -Id $obj.id
        }
    }

}
