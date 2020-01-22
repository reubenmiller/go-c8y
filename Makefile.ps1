[cmdletbinding()]
Param(
    [Parameter(
        Mandatory = $true,
        Position = 0
    )]
    [ValidateSet("update_spec", "build_cli", "build_powershell", "test_powershell")]
    [string[]] $Action
)

foreach ($task in $Action) {
    switch ($task) {
        "update_spec" {
            & "$PSScriptRoot/scripts/generate-spec.ps1";
        }

        "build_cli" {
            & "$PSScriptRoot/scripts/build-cli/build.ps1";
        }

        "build_powershell" {
            & "$PSScriptRoot/scripts/build-powershell/build.ps1";
        }

        "test_powershell" {
            if ($IsLinux -or $IsMacOS) {
                pwsh -File $PSScriptRoot/tools/PSc8y/tests.ps1 -NonInteractive
            } else {
                powershell -File $PSScriptRoot/tools/PSc8y/tests.ps1 -NonInteractive
            }

            if (Get-Command "extent" -ErrorAction SilentlyContinue) {
                Write-Host "Creating html report"
                extent.exe -i $PSScriptRoot/report.xml -o $PSScriptRoot/reports
            }
        }
    }
}
