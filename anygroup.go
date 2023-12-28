package anygroup

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	LogFormat string `long:"log-format" choice:"text" choice:"json" default:"text" required:"false"`
	Verbose   []bool `short:"v" long:"verbose" description:"Show verbose debug information, each -v bumps log level"`
	logLevel  slog.Level
	RootDir   string `short:"r" long:"root" description:"Specify the root directory" default:"."`
}

func Execute() int {
	if err := parseFlags(); err != nil {
		return 1
	}

	if err := setLogLevel(); err != nil {
		return 1
	}

	if err := setupLogger(); err != nil {
		return 1
	}

	if err := run(); err != nil {
		slog.Error("run failed", "error", err)
		return 1
	}

	return 0
}

func parseFlags() error {
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	return err
}

func run() error {
	rootDir := opts.RootDir

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			parseFile(path)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func parseFile(filename string) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("%s: %v\n", filename, err)
		return
	}

	var functionSignatures []string
	var structDefinitions []string
	var variableDefinitions []string

	// Function to process and print if not processed before
	processed := make(map[string]bool)

	processIfNotProcessed := func(s string) {
		if !processed[s] {
			fmt.Printf("%s: %s\n", filename, s)
			processed[s] = true
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			processIfNotProcessed(fmt.Sprintf("function %s", getFunctionSignature(node)))

		case *ast.GenDecl:
			if node.Tok == token.TYPE {
				for _, spec := range node.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						if st, ok := ts.Type.(*ast.StructType); ok {
							processIfNotProcessed(fmt.Sprintf("struct %s", getStructDefinition(ts.Name.Name, st)))
						}
					}
				}
			} else if node.Tok == token.VAR {
				for _, spec := range node.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						processIfNotProcessed(fmt.Sprintf("var %s", getVariableDefinition(vs)))
					}
				}
			}
		}

		return true
	})

	// Print struct definitions first
	for _, structDef := range structDefinitions {
		processIfNotProcessed(structDef)
	}

	// Print variable definitions
	for _, variableDef := range variableDefinitions {
		processIfNotProcessed(variableDef)
	}

	// Print function signatures
	for _, signature := range functionSignatures {
		processIfNotProcessed(signature)
	}
}

func getFunctionSignature(fn *ast.FuncDecl) string {
	funcName := fn.Name.Name

	var params []string
	for _, param := range fn.Type.Params.List {
		for _, name := range param.Names {
			params = append(params, name.Name)
		}
	}

	var results []string
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			if result.Type != nil {
				switch ident := result.Type.(type) {
				case *ast.Ident:
					results = append(results, ident.Name)
				}
			}
		}
	}

	signature := fmt.Sprintf("%s(%s) %s", funcName, strings.Join(params, ", "), strings.Join(results, ", "))
	return signature
}

func getStructDefinition(structName string, st *ast.StructType) string {
	var fields []string
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			fields = append(fields, name.Name)
		}
	}
	return fmt.Sprintf("type %s struct { %s }", structName, strings.Join(fields, ", "))
}

// Function to get variable definition string
func getVariableDefinition(vs *ast.ValueSpec) string {
	var variables []string
	for _, name := range vs.Names {
		variables = append(variables, name.Name)
	}

	if len(variables) == 0 {
		return "" // Don't print anything for variables without names
	}

	var typeStr string
	if vs.Type != nil {
		typeStr = removeTokenInfo(fmt.Sprint(vs.Type))
	}

	var valueStr string
	if len(vs.Values) > 0 {
		valueStr = fmt.Sprintf("%#v", vs.Values)
	}

	return fmt.Sprintf("var %s %s %s", strings.Join(variables, ", "), typeStr, valueStr)
}

// Function to remove token information from string
func removeTokenInfo(s string) string {
	// The format of token information is %!s(token.Type=value)
	// We can remove this part to get a cleaner output
	return strings.TrimPrefix(strings.TrimSuffix(s, ">"), "%!s(")
}
