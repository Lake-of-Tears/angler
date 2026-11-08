package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	SchemaFormatGemini  = "gemini"
	SchemaFormatOpenAI  = "openai"
	SchemaFormatOpenAPI = "openapi"
)

var (
	toolDefsFormat     string
	includeGlobalFlags bool
)

// JSONSchemaProperty represents a single parameter property in JSON Schema.
type JSONSchemaProperty struct {
	Type        string              `json:"type" yaml:"type"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	Default     interface{}         `json:"default,omitempty" yaml:"default,omitempty"`
	Items       *JSONSchemaProperty `json:"items,omitempty" yaml:"items,omitempty"`
}

// JSONSchemaObject represents an object parameter schema for a tool/function.
type JSONSchemaObject struct {
	Type                 string                        `json:"type" yaml:"type"`
	Properties           map[string]JSONSchemaProperty `json:"properties" yaml:"properties"`
	Required             []string                      `json:"required,omitempty" yaml:"required,omitempty"`
	AdditionalProperties *bool                         `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
}

// GeminiFunctionDeclaration follows Google Gemini's function declaration format.
type GeminiFunctionDeclaration struct {
	Name        string           `json:"name" yaml:"name"`
	Description string           `json:"description" yaml:"description"`
	Parameters  JSONSchemaObject `json:"parameters" yaml:"parameters"`
}

// GeminiToolsSpec wraps function declarations for Gemini tool configuration.
type GeminiToolsSpec struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"function_declarations" yaml:"function_declarations"`
}

// OpenAIFunctionDef represents the function payload inside an OpenAI tool.
type OpenAIFunctionDef struct {
	Name        string           `json:"name" yaml:"name"`
	Description string           `json:"description" yaml:"description"`
	Parameters  JSONSchemaObject `json:"parameters" yaml:"parameters"`
}

// OpenAITool represents an OpenAI / Anthropic compatible tool definition.
type OpenAITool struct {
	Type     string            `json:"type" yaml:"type"`
	Function OpenAIFunctionDef `json:"function" yaml:"function"`
}

// OpenAPISpec represents an OpenAPI 3.1.0 document for CLI tool discovery.
type OpenAPISpec struct {
	OpenAPI string                     `json:"openapi" yaml:"openapi"`
	Info    OpenAPIInfo                `json:"info" yaml:"info"`
	Paths   map[string]OpenAPIPathItem `json:"paths" yaml:"paths"`
}

type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

type OpenAPIPathItem struct {
	Post OpenAPIOperation `json:"post" yaml:"post"`
}

type OpenAPIOperation struct {
	OperationID string                    `json:"operationId" yaml:"operationId"`
	Summary     string                    `json:"summary" yaml:"summary"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	RequestBody *OpenAPIRequestBody       `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]OpenAPIRespDef `json:"responses" yaml:"responses"`
}

type OpenAPIRequestBody struct {
	Required bool                          `json:"required" yaml:"required"`
	Content  map[string]OpenAPIMediaObject `json:"content" yaml:"content"`
}

type OpenAPIMediaObject struct {
	Schema JSONSchemaObject `json:"schema" yaml:"schema"`
}

type OpenAPIRespDef struct {
	Description string                        `json:"description" yaml:"description"`
	Content     map[string]OpenAPIMediaObject `json:"content,omitempty" yaml:"content,omitempty"`
}

type introspectedCommand struct {
	ToolName     string
	PathSegments []string
	Summary      string
	Description  string
	Parameters   JSONSchemaObject
}

var toolDefsCmd = &cobra.Command{
	Use:     "tool-defs",
	Aliases: []string{"schema", "tools"},
	Short:   "Export LLM tool calling schemas (Gemini, OpenAI, OpenAPI 3.1)",
	Long: `Introspect the Angler CLI command tree and export standard function/tool definitions
for LLM agent frameworks (Google Gemini, OpenAI, Anthropic, LangChain, AutoGen, CrewAI, Antigravity).

Examples:
  angler tool-defs --format gemini
  angler tool-defs --format openai
  angler schema --format openapi
  angler tool-defs --format openai --output yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaFmt := strings.ToLower(strings.TrimSpace(toolDefsFormat))
		switch schemaFmt {
		case SchemaFormatGemini, SchemaFormatOpenAI, SchemaFormatOpenAPI:
		default:
			return ui.NewUsageError(fmt.Errorf("invalid schema format %q (supported: gemini, openai, openapi)", toolDefsFormat))
		}

		payload, err := GenerateToolSchema(rootCmd, schemaFmt, includeGlobalFlags)
		if err != nil {
			return err
		}

		// Tool definitions are always structured documents: use yaml or toon if explicitly requested via --output, else json.
		outFmt := ui.GetOutputFormat()
		if outFmt != ui.FormatYAML && outFmt != ui.FormatTOON {
			outFmt = ui.FormatJSON
		}

		rendered, err := ui.FormatData(outFmt, payload)
		if err != nil {
			return err
		}
		fmt.Fprintln(ui.OutWriter, rendered)
		return nil
	},
}

// GenerateToolSchema inspects root and builds the requested schema payload (gemini, openai, or openapi).
func GenerateToolSchema(root *cobra.Command, schemaFmt string, includeGlobals bool) (interface{}, error) {
	cmds := collectIntrospectedCommands(root, includeGlobals)

	switch strings.ToLower(schemaFmt) {
	case SchemaFormatGemini:
		decls := make([]GeminiFunctionDeclaration, 0, len(cmds))
		for _, c := range cmds {
			params := c.Parameters
			params.AdditionalProperties = nil
			decls = append(decls, GeminiFunctionDeclaration{
				Name:        c.ToolName,
				Description: c.Description,
				Parameters:  params,
			})
		}
		return GeminiToolsSpec{
			FunctionDeclarations: decls,
		}, nil

	case SchemaFormatOpenAI:
		tools := make([]OpenAITool, 0, len(cmds))
		falseVal := false
		for _, c := range cmds {
			params := c.Parameters
			params.AdditionalProperties = &falseVal
			tools = append(tools, OpenAITool{
				Type: "function",
				Function: OpenAIFunctionDef{
					Name:        c.ToolName,
					Description: c.Description,
					Parameters:  params,
				},
			})
		}
		return tools, nil

	case SchemaFormatOpenAPI:
		paths := make(map[string]OpenAPIPathItem, len(cmds))
		for _, c := range cmds {
			route := "/commands/" + strings.Join(c.PathSegments, "/")
			op := OpenAPIOperation{
				OperationID: c.ToolName,
				Summary:     c.Summary,
				Description: c.Description,
				RequestBody: &OpenAPIRequestBody{
					Required: len(c.Parameters.Required) > 0,
					Content: map[string]OpenAPIMediaObject{
						"application/json": {
							Schema: c.Parameters,
						},
					},
				},
				Responses: map[string]OpenAPIRespDef{
					"200": {
						Description: "Successful command execution result",
						Content: map[string]OpenAPIMediaObject{
							"application/json": {
								Schema: JSONSchemaObject{
									Type:       "object",
									Properties: map[string]JSONSchemaProperty{},
								},
							},
						},
					},
				},
			}
			paths[route] = OpenAPIPathItem{Post: op}
		}

		ver := Version
		if ver == "" {
			ver = "dev"
		}
		return OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: OpenAPIInfo{
				Title:       "Angler CLI Tool Definitions",
				Description: root.Short,
				Version:     ver,
			},
			Paths: paths,
		}, nil

	default:
		return nil, ui.NewUsageError(fmt.Errorf("invalid schema format %q (supported: gemini, openai, openapi)", schemaFmt))
	}
}

func collectIntrospectedCommands(root *cobra.Command, includeGlobals bool) []introspectedCommand {
	var collected []introspectedCommand

	var walk func(cmd *cobra.Command, path []string)
	walk = func(cmd *cobra.Command, path []string) {
		subcommands := cmd.Commands()
		sort.Slice(subcommands, func(i, j int) bool {
			return subcommands[i].Name() < subcommands[j].Name()
		})

		hasRunnableChildren := false
		for _, sub := range subcommands {
			if shouldSkipCommand(sub) {
				continue
			}
			hasRunnableChildren = true
			walk(sub, append(append([]string{}, path...), sub.Name()))
		}

		// Export leaf runnable commands (and runnable commands without subcommands)
		if cmd != root && !hasRunnableChildren && (cmd.Run != nil || cmd.RunE != nil) {
			collected = append(collected, buildIntrospectedCommand(cmd, path, includeGlobals))
		}
	}

	walk(root, nil)
	return collected
}

func shouldSkipCommand(cmd *cobra.Command) bool {
	if cmd.Hidden {
		return true
	}
	switch cmd.Name() {
	case "help", "completion", "tool-defs", "schema", "tools":
		return true
	}
	return false
}

func buildIntrospectedCommand(cmd *cobra.Command, path []string, includeGlobals bool) introspectedCommand {
	sanitizedSegments := make([]string, len(path))
	for i, seg := range path {
		sanitizedSegments[i] = strings.ReplaceAll(seg, "-", "_")
	}
	toolName := "angler_" + strings.Join(sanitizedSegments, "_")

	desc := strings.TrimSpace(cmd.Short)
	if cmd.Long != "" {
		firstPara := strings.TrimSpace(strings.Split(cmd.Long, "\n\n")[0])
		if firstPara != "" && !strings.EqualFold(firstPara, desc) {
			desc = fmt.Sprintf("%s. %s", strings.TrimSuffix(desc, "."), firstPara)
		}
	}

	props := make(map[string]JSONSchemaProperty)
	var required []string

	// 1. Extract positional arguments from cmd.Use (e.g., "use <name>", "query [SQL]", "ask [QUESTION]")
	useParts := strings.Fields(cmd.Use)
	if len(useParts) > 1 {
		for _, token := range useParts[1:] {
			isAngleRequired := strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">")
			cleaned := strings.Trim(token, "<>[]().")
			cleaned = strings.ToLower(strings.ReplaceAll(cleaned, "-", "_"))
			if cleaned == "" || cleaned == "flags" {
				continue
			}

			argDesc := fmt.Sprintf("Positional argument %s for '%s'", token, cmd.CommandPath())
			props[cleaned] = JSONSchemaProperty{
				Type:        "string",
				Description: argDesc,
			}

			// Mark required if enclosed in <...> or if the command requires it and has no flag alternative
			if isAngleRequired || (cmd.Name() == "ask" && cleaned == "question") {
				required = append(required, cleaned)
			}
		}
	}

	// 2. Extract flags
	addFlagToSchema := func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}
		propName := strings.ReplaceAll(f.Name, "-", "_")
		if _, exists := props[propName]; exists {
			propName = "flag_" + propName
		}

		prop := mapFlagToProperty(f)
		props[propName] = prop

		if isFlagRequired(f) {
			required = append(required, propName)
		}
	}

	cmd.NonInheritedFlags().VisitAll(addFlagToSchema)
	if includeGlobals {
		cmd.InheritedFlags().VisitAll(addFlagToSchema)
	}

	sort.Strings(required)

	return introspectedCommand{
		ToolName:     toolName,
		PathSegments: path,
		Summary:      cmd.Short,
		Description:  desc,
		Parameters: JSONSchemaObject{
			Type:       "object",
			Properties: props,
			Required:   required,
		},
	}
}

func mapFlagToProperty(f *pflag.Flag) JSONSchemaProperty {
	flagType := f.Value.Type()
	prop := JSONSchemaProperty{
		Description: f.Usage,
	}

	switch flagType {
	case "bool":
		prop.Type = "boolean"
		if f.DefValue == "true" {
			prop.Default = true
		} else if f.DefValue == "false" {
			prop.Default = false
		}
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "count":
		prop.Type = "integer"
		if v, err := strconv.Atoi(f.DefValue); err == nil && v != 0 {
			prop.Default = v
		}
	case "float32", "float64":
		prop.Type = "number"
		if v, err := strconv.ParseFloat(f.DefValue, 64); err == nil && v != 0 {
			prop.Default = v
		}
	case "stringSlice", "stringArray":
		prop.Type = "array"
		prop.Items = &JSONSchemaProperty{Type: "string"}
	case "intSlice":
		prop.Type = "array"
		prop.Items = &JSONSchemaProperty{Type: "integer"}
	default:
		prop.Type = "string"
		if f.DefValue != "" {
			prop.Default = f.DefValue
		}
	}

	return prop
}

func isFlagRequired(f *pflag.Flag) bool {
	if f.Annotations == nil {
		return false
	}
	if vals, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
		return len(vals) > 0 && vals[0] == "true"
	}
	return false
}

func init() {
	toolDefsCmd.Flags().StringVarP(&toolDefsFormat, "format", "f", SchemaFormatOpenAI, "Schema output format: gemini, openai, openapi")
	toolDefsCmd.Flags().BoolVar(&includeGlobalFlags, "include-global-flags", false, "Include global persistent flags in every tool definition")
	rootCmd.AddCommand(toolDefsCmd)
}
