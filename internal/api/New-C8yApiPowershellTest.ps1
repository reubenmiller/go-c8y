Function New-C8yApiPowershellTest {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true,
            Position = 0
        )]
        [string] $Name,

        [Parameter(
            Mandatory = $true
        )]
        [hashtable[]] $TestCaseVariables,

        [string] $TestCaseTemplateFile,

        [string] $TemplateFile = "powershell/templates/test.template.ps1",

        [Parameter(
            Mandatory = $true
        )]
        [string] $OutFolder
    )

    $Template = Get-Content $TemplateFile -Raw

    $TestCaseTemplate = Get-Content $TestCaseTemplateFile -Raw

    $TestCases = foreach ($TestCase in $TestCaseVariables) {
        $iTestCaseTemplate = "$TestCaseTemplate"
        foreach ($variableName in $TestCase.Keys) {
            $iTestCaseTemplate = $iTestCaseTemplate -replace "{{\s*$variableName\s*}}", $TestCase[$variableName]
        }
        $iTestCaseTemplate
    }

    $Variables = @{
        CmdletName = $Name
        TestCases = $TestCases -join "`n"
        BeforeEach = ""
        AfterEach = ""
    }

    foreach ($variableName in $Variables.Keys) {
        $Template = $Template -replace "{{\s*$variableName\s*}}", $Variables[$variableName]
    }

    $OutFile = Join-Path -Path $OutFolder -ChildPath "${Name}.auto.Test.ps1"

    # Write to file (without BOM)
    $Utf8NoBomEncoding = New-Object System.Text.UTF8Encoding $False
	[System.IO.File]::WriteAllLines($OutFile, $Template, $Utf8NoBomEncoding)
}
