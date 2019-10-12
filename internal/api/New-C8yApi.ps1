Function New-C8yApi {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true,
            Position = 0
        )]
		[string[]] $InFile,

		[Parameter(
            Mandatory = $true,
            Position = 1
        )]
		[string] $OutputDir
	)

	Begin {
		if (!(Test-Path $OutputDir)) {
			$null = New-Item -Type Directory -Path $OutputDir
		}
	}

    Process {
        foreach ($iFile in $InFile) {
			$Path = Resolve-Path $iFile

            $Specifications = Get-Content $Path -Raw | ConvertFrom-Json

			foreach ($iSpec in $Specifications) {
				New-C8yApiGoCommand -Specification $iSpec -OutputDir:$OutputDir
			}
        }
    }
}
