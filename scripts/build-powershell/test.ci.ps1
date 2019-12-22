$ErrorActionPreference = 'Stop'
$ConfirmPreference = "None"

# required in non-interactive mode, otherwise powershell throws errors (regardless of confirmation preference)
$PSDefaultParameterValues = @{"*:Confirm"=$false}

try {

	Import-Module -Name Pester
	$ProjectRoot = "$ENV:APPVEYOR_BUILD_FOLDER/tools/PSc8y"

	$testResultsFilePath = "$ProjectRoot/TestResults.xml"

	$invPesterParams = @{
        Script = "$ProjectRoot/Tests"
		OutputFormat = 'NUnitXml'
		OutputFile = $testResultsFilePath
		EnableExit = $true
	}
	Invoke-Pester @invPesterParams

    if ($env:APPVEYOR) {
        $Address = "https://ci.appveyor.com/api/testresults/nunit/$($env:APPVEYOR_JOB_ID)"
        (New-Object 'System.Net.WebClient').UploadFile( $Address, $testResultsFilePath )
    }
	
} catch {
	Write-Error -Message $_.Exception.Message
	$host.SetShouldExit($LastExitCode)
}