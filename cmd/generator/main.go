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
	providerAnnotationTag  = "@provider"
	decoratorAnnotationTag = "@decorator"
	whenAnnotationTag      = "@when"
	injectAnnotationTag    = "@inject"
	configAnnotationTag    = "@config"
)

type (
	ProviderDefinition struct {
		Named       string
		Description string

		FnName     string
		ImportPath string

		Dependencies []InjectAnnotation
		Priority     int

		Conditions []WhenAnnotation
	}

	DecoratorDefinition struct {
		Decorate    string
		Description string

		FnName     string
		ImportPath string

		Dependencies []InjectAnnotation
		Priority     int

		Conditions []WhenAnnotation
	}

	ConfigDefinition struct {
		TypeName   string
		ImportPath string
		Annotation ConfigAnnotation
	}

	RegistryDefinition struct {
		PackageName string
		StructName  string
	}
)

func (p ProviderDefinition) String() string {
	return fmt.Sprintf(
		`✨ Provider: %s
Description: %s
Import Path: %s
Named: %s
Priority: %d
Dependencies: [%s]`,
		p.FnName,
		p.Description,
		p.ImportPath,
		p.Named,
		p.Priority,
		strings.Join(slices.Map(p.Dependencies, InjectAnnotation.String), ", "),
	)
}

func (d DecoratorDefinition) String() string {
	return fmt.Sprintf(
		`🎨️ Decorator: %s
Description: %s
Import Path: %s
Decorate: %s
Priority: %d
Dependencies: [%s]`,
		d.FnName,
		d.Description,
		d.ImportPath,
		d.Decorate,
		d.Priority,
		strings.Join(slices.Map(d.Dependencies, InjectAnnotation.String), ", "),
	)
}

func (c ConfigDefinition) String() string {
	return fmt.Sprintf(
		`📦 Config: %s
Import Path: %s`,
		c.TypeName,
		c.ImportPath,
	)
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
	dryRun := os.Getenv("DRY_RUN") == "true"

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
	// - functions annotated with @decorator
	// - a struct that embeds godi.EmptyRegistry
	// - struct with @config annotation
	var providerDefinitions []ProviderDefinition
	var decoratorDefinitions []DecoratorDefinition
	var configDefinitions []ConfigDefinition
	var registryDefinition *RegistryDefinition

	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax,
	}
	pkgs, _ := packages.Load(cfg, "./...")

	allPackages := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		allPackages[pkg.PkgPath] = pkg
		allPackages[pkg.ID] = pkg
	}

	for _, pkg := range pkgs {
		logger := logger.With().Str("package", pkg.ID).Logger()
		logger.Debug().Msg("Scanning package")
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
														logger := logger.With().Str("struct", typeSpec.Name.Name).Logger()

														logger.Debug().Msg("=> Found Registry")
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
					if fn.Doc != nil && strings.Contains(fn.Doc.Text(), providerAnnotationTag) {
						logger := logger.With().Str("provider", fn.Name.Name).Logger()

						logger.Debug().Msg("=> Found provider")
						providerAnnotation := parseProviderDecoratorAnnotation(&logger, fn.Name.Name, fn.Doc.Text(), providerAnnotationTag)

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

						dependencies := make([]InjectAnnotation, len(fn.Type.Params.List))
						if fn.Type.Params != nil {
							for idx, param := range fn.Type.Params.List {
								for _, paramName := range param.Names {
									loggerParam := logger.With().Str("param", paramName.Name).Logger()

									dependencies[idx] = parseInjectAnnotation(
										&loggerParam,
										findCommentForParam(pkg.Fset, file, param),
									)
								}
							}
						}

						providerDefinitions = append(providerDefinitions, ProviderDefinition{
							FnName:       fn.Name.Name,
							Description:  providerAnnotation.description,
							ImportPath:   importPath,
							Named:        named,
							Priority:     priority,
							Dependencies: dependencies,
							Conditions:   providerAnnotation.conditions,
						})
					} else if fn.Doc != nil && strings.Contains(fn.Doc.Text(), decoratorAnnotationTag) {
						logger := logger.With().Str("provider", fn.Name.Name).Logger()

						logger.Debug().Msg("=> Found decorator")
						decoratorAnnotation := parseProviderDecoratorAnnotation(&logger, fn.Name.Name, fn.Doc.Text(), decoratorAnnotationTag)

						var (
							decorate string
							priority int
						)
						if n, found := decoratorAnnotation.Named(); found {
							decorate = n
						} else {
							logger.Error().Msgf("Decorator %s must have a named property to name the component being decorated", fn.Name.Name)
							return true
						}
						if p, found := decoratorAnnotation.Priority(); found {
							priority = p
						}

						dependencies := make([]InjectAnnotation, len(fn.Type.Params.List)-1) // skip the first parameter
						if fn.Type.Params != nil {
							for idx, param := range fn.Type.Params.List {
								for _, paramName := range param.Names {
									if idx == 0 {
										// skip the first parameter as it's the component being decorated
										continue
									}
									loggerParam := logger.With().Str("param", paramName.Name).Logger()

									dependencies[idx-1] = parseInjectAnnotation(
										&loggerParam,
										findCommentForParam(pkg.Fset, file, param),
									)
								}
							}
						}

						decoratorDefinitions = append(decoratorDefinitions, DecoratorDefinition{
							FnName:       fn.Name.Name,
							Description:  decoratorAnnotation.description,
							ImportPath:   importPath,
							Decorate:     decorate,
							Priority:     priority,
							Dependencies: dependencies,
							Conditions:   decoratorAnnotation.conditions,
						})
					}
				} else if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
					// look for structs annotated with @config
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if _, ok := typeSpec.Type.(*ast.StructType); ok {
								if genDecl.Doc != nil && strings.Contains(genDecl.Doc.Text(), configAnnotationTag) {
									logger := logger.With().Str("struct", typeSpec.Name.Name).Logger()

									logger.Debug().Msg("=> Found config")

									configDefinitions = append(
										configDefinitions,
										ConfigDefinition{
											TypeName:   typeSpec.Name.Name,
											ImportPath: importPath,
											Annotation: parseConfigAnnotation(&logger, typeSpec.Name.Name, genDecl.Doc.Text()),
										},
									)
								}
							}
						}
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
	logger.Info().Msgf("🎯 %d providers found in the module", len(providerDefinitions))
	definitionsLogs := slices.Map(providerDefinitions, ProviderDefinition.String)
	logger.Debug().Msgf("Providers:\n%s", strings.Join(definitionsLogs, "\n----\n"))
	logger.Info().Msgf("🎯 %d decorators found in the module", len(decoratorDefinitions))
	decoratorDefinitionsLogs := slices.Map(decoratorDefinitions, DecoratorDefinition.String)
	logger.Debug().Msgf("Decorators:\n%s", strings.Join(decoratorDefinitionsLogs, "\n----\n"))
	logger.Info().Msgf("🎯 %d config found in the module", len(configDefinitions))
	configsLogs := slices.Map(configDefinitions, ConfigDefinition.String)
	logger.Debug().Msgf("Configs:\n%s", strings.Join(configsLogs, "\n----\n"))
	logger.Info().Msgf("🕵️‍♂️ Scanning completed in %s", stopScan.Sub(startScan))

	// generate the code
	outputPath := filepath.Join(
		filepath.Dir(targetFilePath),
		strings.TrimSuffix(filepath.Base(targetFilePath), ".go")+"_gen.go",
	)
	if dryRun {
		outputPath = filepath.Join("/tmp", filepath.Base(outputPath))
	}

	err = generateCode(outputPath, registryDefinition, providerDefinitions, decoratorDefinitions, configDefinitions)
	if err != nil {
		logger.Error().Err(err).Msgf("Failed to generate code in %s", outputPath)
		os.Exit(1)
	} else {
		logger.Info().Msgf("✅ Code generated successfully in %s", outputPath)
	}
}
