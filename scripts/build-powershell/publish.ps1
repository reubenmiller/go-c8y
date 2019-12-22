[cmdletbinding()]
Param()
$ErrorActionPreference = 'Stop'

Import-Module PowershellGet -MinimumVersion "2.0.0"

try {
	## Don't upload the build scripts and appveyor.yml to PowerShell Gallery
	$tempName = New-TemporaryFile
	Remove-Item $tempName -Force
	$tempmoduleFolderPath = Join-Path $tempName -ChildPath "PSc8y"
	$null = New-Item $tempmoduleFolderPath -ItemType Directory
	Write-Host "Temp folder: $tempmoduleFolderPath"

	## Remove all of the files/folders to exclude out of the main folder
	$excludeFromPublish = @(
		'PSc8y[\\\/]c8y'
		'PSc8y[\\\/]appveyor\.yml'
		'PSc8y[\\\/]\.git'
		'PSc8y[\\\/]\.nuspec'
		'PSc8y[\\\/]README\.md'
		'PSc8y[\\\/]CHANGELOG\.md'
		'PSc8y[\\\/]tests\.ps1'
		'PSc8y[\\\/]Tests'
	)

	$ProjectDir = Resolve-Path "$PSScriptRoot/../../tools/PSc8y"

	Copy-Item -Path "$ProjectDir/*" -Destination "$tempmoduleFolderPath/" -Recurse

	$exclude = $excludeFromPublish -join '|'
	Get-ChildItem -Path $tempmoduleFolderPath -Recurse `
		| Where-Object { $_.FullName -match $exclude } `
		| Remove-Item -Force -Recurse


	## Publish module to PowerShell Gallery
	$publishParams = @{
		Path        = $tempmoduleFolderPath
		NuGetApiKey = $env:nuget_apikey
	}
	Publish-Module @publishParams

} catch {
	Write-Error -Message $_.Exception.Message
	$host.SetShouldExit($LastExitCode)
}