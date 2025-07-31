package main

import (
	"fmt"
	"github.com/a-peyrard/godi/slices"
	"github.com/rs/zerolog"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	providerAnnotationTag = "@provider"
	injectAnnotationTag   = "@inject"
)

type (
	ProviderDefinition struct {
		Named       string
		Description string

		FnName       string
		Dependencies []string

		Priority int

		ImportPath string
	}

	Dependency struct {
		Named string
	}

	RegistryDefinition struct {
		PackageName string
		StructName  string
	}
)

func (p ProviderDefinition) String() string {
	return fmt.Sprintf(`✨ Provider: %s
Description: %s
Import Path: %s
Named: %s
Priority: %d
Dependencies: [%s]`, p.FnName, p.Description, p.ImportPath, p.Named, p.Priority, strings.Join(p.Dependencies, ", "))
}

func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatType(t.X)
	case *ast.SelectorExpr:
		return formatType(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

func findCommentForParam(fset *token.FileSet, file *ast.File, param *ast.Field) string {
	paramLine := fset.Position(param.Pos()).Line

	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			commentLine := fset.Position(comment.Pos()).Line
			if commentLine == paramLine {
				return comment.Text
			}
		}
	}
	return ""
}

func findModuleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
	return "."
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime}).
		With().
		Timestamp().
		Logger()

	startScan := time.Now()

	// capture the target file/package, where the generator is invoked
	targetFile := os.Getenv("GOFILE")
	targetPackage := os.Getenv("GOPACKAGE")
	currentDir, _ := os.Getwd()
	targetFilePath := filepath.Join(currentDir, targetFile)

	// no switch to the root of the module as we want to be able to scan the whole module
	moduleRoot := findModuleRoot()
	err := os.Chdir(moduleRoot)
	if err != nil {
		log.Fatalf("Failed to change directory to module root: %v\n", err)
	}

	// analyze all the packages in the module
	// we are looking for multiple things:
	// - functions annotated with @provider
	// - a struct that embeds godi.EmptyRegistry
	var definitions []ProviderDefinition
	var registryDefinition *RegistryDefinition

	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax,
	}
	pkgs, _ := packages.Load(cfg, "./...")

	for _, pkg := range pkgs {
		logger.Debug().Msgf("scanning package '%s'", pkg.ID)

		for _, file := range pkg.Syntax {
			filePath := pkg.Fset.Position(file.Pos()).Filename
			packageName := file.Name.Name
			importPath := pkg.ID

			// only look for Registry struct in the file triggering the generation
			if filePath == targetFilePath {
				// Look for struct embedding godi.EmptyRegistry
				ast.Inspect(file, func(n ast.Node) bool {
					if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
						for _, spec := range genDecl.Specs {
							if typeSpec, ok := spec.(*ast.TypeSpec); ok {
								if structType, ok := typeSpec.Type.(*ast.StructType); ok {
									for _, field := range structType.Fields.List {
										if len(field.Names) == 0 { // Embedded field
											if sel, ok := field.Type.(*ast.SelectorExpr); ok {
												if ident, ok := sel.X.(*ast.Ident); ok {
													if ident.Name == "godi" && sel.Sel.Name == "EmptyRegistry" {
														logger.Debug().Msgf("=> Found Registry struct: %s in package %s",
															typeSpec.Name.Name, packageName)
														registryDefinition = &RegistryDefinition{
															PackageName: packageName,
															StructName:  typeSpec.Name.Name,
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
					return true
				})
			}

			// look for @provider functions
			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if fn.Doc != nil && strings.Contains(fn.Doc.Text(), "@provider") {
						logger := logger.With().Str("provider", fn.Name.Name).Logger()

						logger.Debug().Msgf("=> Found provider in %s", importPath)
						providerAnnotation := parseProviderAnnotation(&logger, fn.Doc.Text())

						var (
							named    string
							priority int
						)
						if n, found := providerAnnotation.Named(); found {
							named = n
						}
						if p, found := providerAnnotation.Priority(); found {
							priority = p
						}

						var dependencies []string
						if fn.Type.Params != nil {
							for _, param := range fn.Type.Params.List {
								for _, paramName := range param.Names {
									loggerParam := logger.With().Str("param", paramName.Name).Logger()

									comment := findCommentForParam(pkg.Fset, file, param)
									var depName string
									if comment != "" && strings.Contains(comment, injectAnnotationTag) {
										injectAnnotation := parseInjectAnnotation(&loggerParam, comment)
										if n, found := injectAnnotation.Named(); found {
											depName = n
										}
									}
									dependencies = append(dependencies, depName)
								}
							}
						}

						definitions = append(definitions, ProviderDefinition{
							FnName:       file.Name.Name + "." + fn.Name.Name,
							Description:  providerAnnotation.description,
							ImportPath:   importPath,
							Named:        named,
							Priority:     priority,
							Dependencies: dependencies,
						})
					}
				}
				return true
			})
		}
	}

	stopScan := time.Now()

	if registryDefinition == nil {
		logger.Error().Msgf("No Registry struct found in the target package: %s, make sure you have a struct like this:\ntype Registry {\n    godi.EmptyRegistry\n}", targetPackage)
		os.Exit(1)
	}

	logger.Info().Msgf("👨‍🔧 Registry found: %+v", registryDefinition)
	logger.Info().Msgf("🎯 %d providers found in the module", len(definitions))
	definitionsLogs := slices.Map(definitions, ProviderDefinition.String)
	logger.Debug().Msgf("Providers:\n%s", strings.Join(definitionsLogs, "\n----\n"))
	logger.Info().Msgf("🕵️‍♂️ Scanning completed in %s", stopScan.Sub(startScan))
}
