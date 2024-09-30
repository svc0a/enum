package gen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

func Generate(filename string) {
	// 解析文件AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		return
	}

	// 创建一个新的声明列表，保存现有的和新生成的代码
	newDecls := make([]ast.Decl, 0, len(file.Decls))

	// 遍历AST，寻找标注了 @enumGenerated 的类型
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		newDecls = append(newDecls, decl) // 将原有的声明追加到新声明列表
		if !ok || genDecl.Doc == nil {
			continue
		}

		for _, comment := range genDecl.Doc.List {
			if strings.Contains(comment.Text, "@enumGenerated") {
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok {
						enumType := typeSpec.Name.Name
						fmt.Printf("Found enum type: %s\n", enumType)

						// 收集该类型的常量
						values := collectEnumValues(file, enumType)
						fmt.Printf("Enum values: %v\n", values)

						// 检查是否已经定义了 Values 和 String 方法
						hasValuesMethod, hasStringMethod := checkExistingMethods(file, enumType)

						// 生成新的方法
						valuesMethod := generateValuesMethodAST(enumType, values)
						stringMethod := generateStringMethodAST(enumType)

						// 替换或插入 Values 方法
						if hasValuesMethod {
							fmt.Printf("Replacing existing Values method for type %s\n", enumType)
							replaceMethod(file, enumType, "Values", valuesMethod)
						} else {
							newDecls = append(newDecls, valuesMethod)
						}

						// 替换或插入 String 方法
						if hasStringMethod {
							fmt.Printf("Replacing existing String method for type %s\n", enumType)
							replaceMethod(file, enumType, "String", stringMethod)
						} else {
							newDecls = append(newDecls, stringMethod)
						}
					}
				}
			}
		}
	}

	// 用新声明列表替换原文件的声明
	file.Decls = newDecls

	// 将修改后的AST写回文件
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer f.Close()

	if err := printer.Fprint(f, fset, file); err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("Code generation completed successfully!")
}

// 收集与枚举类型相关的常量
func collectEnumValues(file *ast.File, enumType string) []string {
	var values []string

	ast.Inspect(file, func(n ast.Node) bool {
		valueSpec, ok := n.(*ast.ValueSpec)
		if !ok || len(valueSpec.Names) == 0 {
			return true
		}

		// 检查常量的类型是否为目标枚举类型
		if ident, ok := valueSpec.Type.(*ast.Ident); ok && ident.Name == enumType {
			for _, name := range valueSpec.Names {
				values = append(values, name.Name)
			}
		}

		return true
	})

	return values
}

// 检查是否存在 Values 和 String 方法
func checkExistingMethods(file *ast.File, enumType string) (bool, bool) {
	hasValuesMethod := false
	hasStringMethod := false

	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			return true
		}

		// 检查接收者的类型是否匹配
		if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.Ident); ok && starExpr.Name == enumType {
			switch funcDecl.Name.Name {
			case "Values":
				hasValuesMethod = true
			case "String":
				hasStringMethod = true
			}
		}

		return true
	})

	return hasValuesMethod, hasStringMethod
}

// 替换已有方法
func replaceMethod(file *ast.File, enumType string, methodName string, newMethod *ast.FuncDecl) {
	for i, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}

		// 找到同名方法并替换
		if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.Ident); ok && starExpr.Name == enumType && funcDecl.Name.Name == methodName {
			file.Decls[i] = newMethod
			return
		}
	}
}

// 使用 AST 生成 Values() 方法
func generateValuesMethodAST(enumType string, values []string) *ast.FuncDecl {
	// 创建返回的数组类型：[]<enumType>
	returnType := &ast.ArrayType{
		Elt: &ast.Ident{
			Name: enumType,
		},
	}

	// 创建 return 语句
	valueList := make([]ast.Expr, len(values))
	for i, v := range values {
		valueList[i] = &ast.Ident{Name: v}
	}

	returnStmt := &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CompositeLit{
				Type: returnType,
				Elts: valueList,
			},
		},
	}

	// 创建函数体
	body := &ast.BlockStmt{
		List: []ast.Stmt{returnStmt},
	}

	// 创建函数声明
	funcDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: "Values"},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						{Name: "g"},
					},
					Type: &ast.Ident{
						Name: enumType,
					},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.ArrayType{
							Elt: &ast.Ident{
								Name: enumType,
							},
						},
					},
				},
			},
		},
		Body: body,
	}

	return funcDecl
}

// 使用 AST 生成 String() 方法
func generateStringMethodAST(enumType string) *ast.FuncDecl {
	// 创建函数体：return fmt.Sprintf("%v", g)
	returnStmt := &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "fmt"},
					Sel: &ast.Ident{Name: "Sprintf"},
				},
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: "\"%v\"",
					},
					&ast.Ident{Name: "g"},
				},
			},
		},
	}

	// 创建函数体
	body := &ast.BlockStmt{
		List: []ast.Stmt{returnStmt},
	}

	// 创建函数声明
	funcDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: "String"},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						{Name: "g"},
					},
					Type: &ast.Ident{
						Name: enumType,
					},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.Ident{
							Name: "string",
						},
					},
				},
			},
		},
		Body: body,
	}

	return funcDecl
}
