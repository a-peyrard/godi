package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/packages"
	"strings"
)

type FieldAnalysis struct {
	Path       string // "Environment", "Broker.Uris", "Broker.Sasl.Username"
	ImportPath string // "config", "github.com/example/project/pkg/config"
	TypeName   string // "string", "[]string", "*SaslConfig"
}

func analyzeConfigStruct(pkg *packages.Package, structType *ast.StructType) []FieldAnalysis {
	var allFields []FieldAnalysis

	analyzeStructRecursive(pkg, "", structType, &allFields, make(map[string]bool))
	return allFields
}

func analyzeStructRecursive(pkg *packages.Package, prefix string, structType *ast.StructType, results *[]FieldAnalysis, visited map[string]bool) {
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fieldPath := name.Name
			if prefix != "" {
				fieldPath = prefix + "." + name.Name
			}

			// Extract package and type information separately
			packageName, typeName := extractPackageAndType(field.Type, pkg)

			*results = append(*results, FieldAnalysis{
				Path:       fieldPath,
				ImportPath: packageName,
				TypeName:   typeName,
			})

			if _, ok := field.Type.(*ast.StarExpr); ok {
				if nestedPkg, nestedStruct := findNestedStruct(pkg, field.Type); nestedStruct != nil {
					structTypeName := getStructTypeName(field.Type)
					if !visited[structTypeName] {
						visited[structTypeName] = true
						analyzeStructRecursive(nestedPkg, fieldPath, nestedStruct, results, visited)
						delete(visited, structTypeName)
					}
				}
			}
		}
	}
}

func extractPackageAndType(expr ast.Expr, currentPkg *packages.Package) (packageName, typeName string) {
	return extractPackageAndTypeInternal(expr, currentPkg, false)
}

func extractPackageAndTypeInternal(expr ast.Expr, currentPkg *packages.Package, capturePkg bool) (packageName, typeName string) {
	switch t := expr.(type) {
	case *ast.Ident:
		pkgName := ""
		if capturePkg {
			pkgName = currentPkg.ID
		}
		return pkgName, t.Name

	case *ast.StarExpr:
		// Pointer types: *config.BrokerConfig, *SaslConfig
		pkg, typ := extractPackageAndTypeInternal(t.X, currentPkg, true)
		return pkg, "*" + typ

	case *ast.SelectorExpr:
		// Qualified types: config.BrokerConfig
		if ident, ok := t.X.(*ast.Ident); ok {
			packageAlias := ident.Name
			typeName := t.Sel.Name

			// Find the full import path for this package alias
			importPath := findImportPathForAlias(currentPkg, packageAlias)
			return importPath, typeName
		}

	case *ast.ArrayType:
		// Array/slice types: []string, []config.Something
		pkg, typ := extractPackageAndType(t.Elt, currentPkg)
		return pkg, "[]" + typ

	case *ast.MapType:
		// Map types: map[string]int, map[string]config.Something
		keyPkg, keyType := extractPackageAndType(t.Key, currentPkg)
		valPkg, valType := extractPackageAndType(t.Value, currentPkg)

		// For maps, we need to handle multiple packages - this is complex
		// For now, return the value package if key is primitive
		if keyPkg == "" {
			return valPkg, "map[" + keyType + "]" + valType
		}
		// If both have packages, this gets complicated - you might need a different approach
		return "", "map[" + keyType + "]" + valType

	case *ast.ChanType:
		pkg, typ := extractPackageAndType(t.Value, currentPkg)
		return pkg, "chan " + typ

	case *ast.InterfaceType:
		return "", "interface{}"
	}

	return "", "unknown"
}

func findImportPathForAlias(pkg *packages.Package, packageAlias string) string {
	for _, file := range pkg.Syntax {
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			var alias string
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				parts := strings.Split(importPath, "/")
				alias = parts[len(parts)-1]
			}

			if alias == packageAlias {
				return importPath
			}
		}
	}
	return ""
}

func findNestedStruct(pkg *packages.Package, fieldType ast.Expr) (*packages.Package, *ast.StructType) {
	// Handle pointer types: *config.BrokerConfig
	if starExpr, ok := fieldType.(*ast.StarExpr); ok {
		fieldType = starExpr.X
	}

	var typeName string
	if selectorExpr, ok := fieldType.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			typeName = selectorExpr.Sel.Name
			return findStructInPackage(pkg, ident.Name, typeName)
		}
	} else if ident, ok := fieldType.(*ast.Ident); ok {
		typeName = ident.Name
		result := findStructInCurrentPackage(pkg, typeName)
		return pkg, result
	}

	fmt.Printf("DEBUG: Could not parse field type: %T\n", fieldType)
	return nil, nil
}

func findStructInCurrentPackage(pkg *packages.Package, typeName string) *ast.StructType {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == typeName {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								return structType
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func getStructTypeName(fieldType ast.Expr) string {
	if starExpr, ok := fieldType.(*ast.StarExpr); ok {
		fieldType = starExpr.X
	}

	if selectorExpr, ok := fieldType.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			return ident.Name + "." + selectorExpr.Sel.Name
		}
	} else if ident, ok := fieldType.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}

func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatType(t.X)
	case *ast.SelectorExpr:
		return formatType(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + formatType(t.Elt)
	case *ast.MapType:
		return "map[" + formatType(t.Key) + "]" + formatType(t.Value)
	case *ast.ChanType:
		return "chan " + formatType(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}

func findStructInPackage(currentPkg *packages.Package, packageAlias, typeName string) (*packages.Package, *ast.StructType) {
	// Find the import that matches the package alias
	var targetImportPath string

	for _, file := range currentPkg.Syntax {
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// Check if this import matches our package alias
			var alias string
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				// Default alias is the last part of the import path
				parts := strings.Split(importPath, "/")
				alias = parts[len(parts)-1]
			}

			if alias == packageAlias {
				targetImportPath = importPath
				break
			}
		}
		if targetImportPath != "" {
			break
		}
	}

	if targetImportPath == "" {
		return nil, nil
	}

	// Load the target package
	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax,
	}

	targetPkgs, err := packages.Load(cfg, targetImportPath)
	if err != nil || len(targetPkgs) == 0 {
		return nil, nil
	}

	targetPkg := targetPkgs[0]

	// Find the struct in the target package
	return targetPkg, findStructInCurrentPackage(targetPkg, typeName)
}
