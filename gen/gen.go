package gen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

// Generate 解析并生成代码
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

	// 使用缓冲区捕获打印的代码
	var buf bytes.Buffer

	// 打印AST到缓冲区
	err = printer.Fprint(&buf, fset, file)
	if err != nil {
		fmt.Println("Error writing buffer:", err)
		return
	}

	// 使用 go/format 格式化代码
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Println("Error formatting code:", err)
		return
	}

	// 将格式化后的代码写入文件
	if _, err := f.Write(formattedCode); err != nil {
		fmt.Println("Error writing formatted code:", err)
		return
	}

	fmt.Println("Code generation completed successfully!")
}

// collectEnumValues 收集枚举类型的常量值
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

// checkExistingMethods 检查是否存在 Values 和 String 方法
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

// replaceMethod 替换已有方法
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

// generateValuesMethodAST 使用 AST 生成 Values() 方法
func generateValuesMethodAST(enumType string, values []string) *ast.FuncDecl {
	// 创建返回的数组类型：[]string
	returnType := &ast.ArrayType{
		Elt: &ast.Ident{
			Name: "string",
		},
	}

	// 创建 return 语句，调用每个枚举值的 String() 方法
	valueList := make([]ast.Expr, len(values))
	for i, v := range values {
		valueList[i] = &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: v},
				Sel: &ast.Ident{Name: "String"},
			},
			Args: []ast.Expr{},
		}
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
								Name: "string",
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

// generateStringMethodAST 使用 AST 生成 String() 方法
func generateStringMethodAST(enumType string) *ast.FuncDecl {
	// 创建函数体：return string(g)
	returnStmt := &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CallExpr{
				Fun:  &ast.Ident{Name: "string"},
				Args: []ast.Expr{&ast.Ident{Name: "g"}},
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
