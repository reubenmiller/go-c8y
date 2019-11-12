Function New-C8yApiGoCommand {
    [cmdletbinding()]
    Param(
        [Parameter(
            Position = 0,
            ValueFromPipeline = $true,
            ValueFromPipelineByPropertyName = $true,
            Mandatory = $true
        )]
        [object[]] $Specification,

        [string] $OutputDir = "./"
    )

    $Name = $Specification.name
	$NameCamel = $Name[0].ToString().ToUpperInvariant() + $Name.Substring(1)
	$File = Join-Path -Path $OutputDir -ChildPath ("{0}Cmd.go" -f $Name)


    #
    # Meta information
    #
    $Use = $Specification.alias.go
    $Description = $Specification.description
    $DescriptionLong = $Specification.descriptionLong
    $Examples = foreach ($iExample in $Specification.examples.go) {
        $iExample
    }

    #
    # Arguments
    #
    $ArgumentSources = New-Object System.Collections.ArrayList

    if ($Specification.pathParameters) {
        $null = $ArgumentSources.AddRange(([array]$Specification.pathParameters))
    }

    if ($Specification.queryParameters) {
        $null = $ArgumentSources.AddRange(([array]$Specification.queryParameters))
    }
    if ($Specification.body) {
        $null = $ArgumentSources.AddRange(([array]$Specification.body))
    }

    $CommandArgs = foreach ($iArg in $ArgumentSources) {
        $ArgParams = @{
            Name = $iArg.name
            Type = $iArg.type
            OptionName = $iArg.alias
            Description = $iArg.description
            Default = $iArg.default
            Required = $iArg.required
        }
        Get-C8yGoArgs @ArgParams
    }

    $RESTPath = $Specification.path
    $RESTMethod = $Specification.method

    #
    # Body
    #
    $RESTBodyBuilder = New-Object System.Text.StringBuilder
    if ($Specification.body) {
        $null = $RESTBodyBuilder.AppendLine('body.SetMap(getDataFlag(cmd))')

        foreach ($iArg in $Specification.body) {
            $code = New-C8yApiGoGetValueFromFlag -Parameters $iArg -SetterType "body"
            if ($code) {
                $null = $RESTBodyBuilder.AppendLine($code)
            }
        }
    }

    #
    # Path Parameters
    #
    $RESTPathBuilder = New-Object System.Text.StringBuilder
    foreach ($Properties in $Specification.pathParameters) {
        $code = New-C8yApiGoGetValueFromFlag -Parameters $Properties -SetterType "path"
        if ($code) {
            $null = $RESTPathBuilder.AppendLine($code)
        }
    }

    #
    # Query parameters
    #
    $RESTQueryBuilder = New-Object System.Text.StringBuilder
    $null = $RESTQueryBuilder.AppendLine('query := url.Values{}')
    if ($Specification.queryParameters) {
        foreach ($Properties in $Specification.queryParameters) {
            $code = New-C8yApiGoGetValueFromFlag -Parameters $Properties -SetterType "query"
            if ($code) {
                $null = $RESTQueryBuilder.AppendLine($code)
            }
        }
    }

    #
    # Add common options
    #
    $null = $RESTQueryBuilder.AppendLine(@"
    if cmd.Flags().Changed("pageSize") {
        if v, err := cmd.Flags().GetInt("pageSize"); err == nil && v > 0 {
            query.Add("pageSize", fmt.Sprintf("%d", v))
        }
    }

    if cmd.Flags().Changed("withTotalPages") {
        if v, err := cmd.Flags().GetBool("withTotalPages"); err == nil && v {
            query.Add("withTotalPages", "true")
        }
    }
"@)
    #
    # Encode query parameters to a string
    #
    $null = $RESTQueryBuilder.AppendLine(@"
    queryValue, err := url.QueryUnescape(query.Encode())

    if err != nil {
        return newSystemError("Invalid query parameter")
    }
"@)


    #
    # Template
    #
    $Template = @"
// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/url"

    "github.com/fatih/color"
    "github.com/reubenmiller/go-c8y/pkg/mapbuilder"
    "github.com/reubenmiller/go-c8y/pkg/c8y"
    "github.com/spf13/cobra"
    "github.com/tidwall/pretty"
)

type ${Name}Cmd struct {
    *baseCmd
}

func new${NameCamel}Cmd() *${Name}Cmd {
	ccmd := &${Name}Cmd{}

	cmd := &cobra.Command{
		Use:   "$Use",
		Short: "$Description",
		Long:  ``$DescriptionLong``,
        Example: ``
        $($Examples -join "`n`n")
		``,
		RunE: ccmd.${Name},
    }

    cmd.SilenceUsage = true

    $($CommandArgs.SetFlag -join "`n	")

    // Required flags
    $($CommandArgs.Required -join "`n	")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *${Name}Cmd) ${Name}(cmd *cobra.Command, args []string) error {

    // query parameters
    queryValue := url.QueryEscape("")
    $RESTQueryBuilder

    // body
    body := mapbuilder.NewMapBuilder()
    $RESTBodyBuilder

    // path parameters
    pathParameters := make(map[string]string)
    $RESTPathBuilder
    path := replacePathParameters("${RESTPath}", pathParameters)

    // filter and selectors
    filters := getFilterFlag(cmd, "filter")

    return n.do${NameCamel}("${RESTMethod}", path, queryValue, body.GetMap(), filters)
}

func (n *${Name}Cmd) do${NameCamel}(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
    resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
            Path:         path,
            Query:        query,
            Body:         body,
            IgnoreAccept: false,
            DryRun:       globalFlagDryRun,
		})

    if err != nil {
        color.Set(color.FgRed, color.Bold)
    }

    if resp != nil && resp.JSONData != nil {
        // estimate size based on utf8 encoding. 1 char is 1 byte
	    log.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

        var responseText []byte

        if filters != nil && !globalFlagRaw {
			responseText = filters.Apply(*resp.JSONData, "$($Specification.listProperty)")
		} else {
			responseText = []byte(*resp.JSONData)
		}

        if globalFlagPrettyPrint && json.Valid(responseText) {
            fmt.Printf("%s", pretty.Pretty(responseText))
        } else {
            fmt.Printf("%s", responseText)
        }
    }

    color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
"@

	# Must not include BOM!
	$Utf8NoBomEncoding = New-Object System.Text.UTF8Encoding $False
	[System.IO.File]::WriteAllLines($File, $Template, $Utf8NoBomEncoding)

	# Auto format code
	& gofmt -w $File
}

Function Get-C8yGoArgs {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true,
            Position = 0
        )]
        [string] $Name,

        [Parameter(
            Mandatory = $true,
            Position = 1
        )]
        [string] $Type,

        [string] $Required,

        [string] $OptionName,

        [string] $Description,

        [string] $Default
    )

    if ($Required -match "true|yes") {
        $Description = "${Description} (required)"
    }

    $Entry = switch -Regex ($Type) {
        "id" {
            @{
                SetFlag = "addIDFlag(cmd)"
            }
        }

        "json" {
            @{
                SetFlag = "addDataFlag(cmd)"
            }
        }

        "date(from|to|time)" {
            $SetFlag = if ($UseOption) {
                'cmd.Flags().StringP("{0}", "{1}", "{2}", "{3}{4}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().String("{0}", "{1}", "{2}")' -f $Name, $Default, $Description
            }
            @{
                SetFlag = $SetFlag
            }
        }

        "\[\]string" {
            $SetFlag = if ($UseOption) {
                "cmd.Flags().StringSlice(`"${Name}`", `"${OptionName}`", []string{`"${Default}`"}, `"${Description}`")"
            } else {
                "cmd.Flags().StringSlice(`"${Name}`", []string{`"${Default}`"}, `"${Description}`")"
            }
            @{
                SetFlag = $SetFlag
            }
        }

        "\[\]device" {
            $SetFlag = if ($UseOption) {
                "cmd.Flags().StringSliceP(`"${Name}`", []string{`"${Default}`"}, `"${OptionName}`", `"${Description}`")"
            } else {
                "cmd.Flags().StringSlice(`"${Name}`", []string{`"${Default}`"}, `"${Description}`")"
            }

            @{
                SetFlag = $SetFlag
            }
        }

        "^application$" {
            $SetFlag = if ($UseOption) {
                'cmd.Flags().StringP("{0}", "{1}", "{2}", "{3}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().String("{0}", "{1}", "{2}")' -f $Name, $Default, $Description
            }
            @{
                SetFlag = $SetFlag
            }
        }

        "^string$" {
            $SetFlag = if ($UseOption) {
                'cmd.Flags().StringP("{0}", "{1}", "{2}", "{3}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().String("{0}", "{1}", "{2}")' -f $Name, $Default, $Description
            }

            @{
                SetFlag = $SetFlag
            }
        }

        "^tenant$" {
            $SetFlag = if ($UseOption) {
                'cmd.Flags().StringP("{0}", "{1}", "{2}", "{3}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().String("{0}", "{1}", "{2}")' -f $Name, $Default, $Description
            }

            @{
                SetFlag = $SetFlag
            }
        }

        "boolean" {
            if (!$Default) {
                $Default = "false"
            }
            $SetFlag = if ($UseOption) {
                'cmd.Flags().BoolP("{0}", "{1}", {2}, "{3}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().Bool("{0}", {1}, "{2}")' -f $Name, $Default, $Description
            }

            @{
                SetFlag = $SetFlag
            }
        }
    }

    # Set required flag
    if ($Required -match "true|yes") {
        $Entry | Add-Member -MemberType NoteProperty -Name "Required" -Value "cmd.MarkFlagRequired(`"${Name}`")"
        # $Entry.Required = "cmd.MarkFlagRequired(`"${Name}`")"
    }

    $Entry
}

