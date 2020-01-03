[cmdletbinding()]
Param()
$ErrorActionPreference = 'Stop'

# PowerShellGet 2.2.3 required to run correctly on MacOS
try {
	$PowerShellGetVersion = Get-Module -Name PowerShellGet -ListAvailable | ForEach-Object { [version] $_.Version } | Sort-Object -Descending | Select-Object -First 1

	if ($PowerShellGetVersion -lt ([version] "2.2.3")) {
		Install-Module PowerShellGet -MinimumVersion "2.2.3" -Force
		Remove-Module PowerShellGet -Force
		Start-Sleep -Seconds 2
		Import-Module PowerShellGet -MinimumVersion "2.2.3"
	}
} catch {
	Write-Host "PowerShellGet modules"
	Get-Module -Name PowerShellGet -ListAvailable

	$Versions = Get-Module -Name PowerShellGet | Select-Object -ExpandProperty Version
	Write-Host ("Current loaded version: " -f ($Versions -join ","))
}


if ($env:APPVEYOR) {
	& $PSScriptRoot/wait-for-jobs.ps1
}

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
		'PSc8y[\\\/]Dependencies[\\\/]\.gitkeep'
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

	#
	# Build binaries
	#
	$DependenciesDir = "$tempmoduleFolderPath/Dependencies/"
	& $PSScriptRoot/../build-cli/build-binary.ps1 -OutputDir $DependenciesDir -All

	[array] $c8ybinaries = Get-ChildItem -Path $DependenciesDir -Filter "*c8y*"

	if ($c8ybinaries.Count -ne 3) {
		Write-Error "Failed to find all 3 c8y binaries"
		Exit 1
	}

	Write-Host "Publishing module"
	## Publish module to PowerShell Gallery
	$publishParams = @{
		Path        = $tempmoduleFolderPath
		NuGetApiKey = $env:nuget_apikey
		Verbose = $true
	}
	Publish-Module @publishParams

} catch {
	Write-Error -Message $_.Exception.Message
	$host.SetShouldExit($LastExitCode)
}
