Function New-C8yPowershellApi {
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
        $importStatements = foreach ($iFile in $InFile) {
			$Path = Resolve-Path $iFile

            $Specification = Get-Content $Path -Raw | ConvertFrom-Json

			foreach ($iSpec in $Specification.endpoints) {
                New-C8yApiPowershellCommand `
                    -Specification:$iSpec `
                    -Noun $Specification.information.name `
                    -OutputDir:$OutputDir
			}
        }

        $importStatements
    }
}
