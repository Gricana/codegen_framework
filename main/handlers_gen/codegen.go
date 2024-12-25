package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"
)

// ApiSpec описывает метаинформацию о методе API
type ApiSpec struct {
	URL        string `json:"url"`
	Auth       bool   `json:"auth"`
	Method     string `json:"method"`
	MethodName string
}

// Field описывает поле структуры
type Field struct {
	Name      string
	Type      string
	Validator map[string]string
}

// parseTag парсит тег в формате `apivalidator:"key=value,key2=value2"`
func parseTag(tag string) map[string]string {
	parts := strings.Split(tag, ",")
	result := make(map[string]string)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		} else {
			result[kv[0]] = ""
		}
	}
	return result
}

// parseStructFields разбирает поля структуры
func parseStructFields(node *ast.StructType) []Field {
	var fields []Field
	for _, field := range node.Fields.List {
		if len(field.Names) == 0 || field.Tag == nil {
			continue
		}
		name := field.Names[0].Name
		fieldType := ""
		if ident, ok := field.Type.(*ast.Ident); ok {
			fieldType = ident.Name
		}
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		validator := parseTag(tag.Get("apivalidator"))
		fields = append(fields, Field{Name: name, Type: fieldType, Validator: validator})
	}
	return fields
}

// generateValidation генерирует код валидации параметров
func generateValidation(out *strings.Builder, fields []Field) {
	for _, field := range fields {
		// required validation
		if field.Validator["required"] != "" {
			fmt.Fprintf(out, "\tif params.%s == \"\" {\n", field.Name)
			fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusBadRequest)\n")
			fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf(\"%s must be not empty\")})\n", field.Name)
			fmt.Fprintf(out, "\t\treturn\n")
			fmt.Fprintf(out, "\t}\n")
		}

		// min length validation
		if min, ok := field.Validator["min"]; ok {
			if field.Type == "int" {
				fmt.Fprintf(out, "\tif params.%s < %s {\n", field.Name, min)
				fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusBadRequest)\n")
				fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf(\"%s must be >= %s\")})\n", field.Name, min)
				fmt.Fprintf(out, "\t\treturn\n")
				fmt.Fprintf(out, "\t}\n")
			} else if field.Type == "string" {
				fmt.Fprintf(out, "\tif len(params.%s) < %s {\n", field.Name, min)
				fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusBadRequest)\n")
				fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf(\"%s length must be >= %s\")})\n", field.Name, min)
				fmt.Fprintf(out, "\t\treturn\n")
				fmt.Fprintf(out, "\t}\n")
			}
		}

		// max length validation
		if max, ok := field.Validator["max"]; ok {
			if field.Type == "int" {
				fmt.Fprintf(out, "\tif params.%s > %s {\n", field.Name, max)
				fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusBadRequest)\n")
				fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf(\"%s must be <= %s\")})\n", field.Name, max)
				fmt.Fprintf(out, "\t\treturn\n")
				fmt.Fprintf(out, "\t}\n")
			} else if field.Type == "string" {
				fmt.Fprintf(out, "\tif len(params.%s) > %s {\n", field.Name, max)
				fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusBadRequest)\n")
				fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf(\"%s length must be <= %s\")})\n", field.Name, max)
				fmt.Fprintf(out, "\t\treturn\n")
				fmt.Fprintf(out, "\t}\n")
			}
		}

		// enum validation
		if enum, ok := field.Validator["enum"]; ok {
			fmt.Fprintf(out, "\tswitch params.%s {\n", field.Name)
			for _, value := range strings.Split(enum, "|") {
				fmt.Fprintf(out, "\tcase \"%s\":\n", value)
			}
			fmt.Fprintf(out, "\tdefault:\n")
			fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusBadRequest)\n")
			fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf(\"%s must be one of [%s]\")})\n", field.Name, enum)
			fmt.Fprintf(out, "\t\treturn\n")
			fmt.Fprintf(out, "\t}\n")
		}
	}
}

// generateHandler генерирует обработчик метода API
func generateHandler(out *os.File, structName, methodName string, spec ApiSpec, paramType string, fields []Field) {
	fmt.Fprintf(out, "func (h *%s) handler%s(w http.ResponseWriter, r *http.Request) {\n", structName, methodName)

	// Авторизация
	if spec.Auth {
		fmt.Fprintf(out, "\tif r.Header.Get(\"Authorization\") != \"100500\" {\n")
		fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusUnauthorized)\n")
		fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusUnauthorized, Err: fmt.Errorf(\"unauthorized\")})\n")
		fmt.Fprintf(out, "\t\treturn\n")
		fmt.Fprintf(out, "\t}\n")
	}

	// Декодирование параметров
	fmt.Fprintf(out, "\tvar params %s\n", paramType)
	fmt.Fprintf(out, "\tif err := json.NewDecoder(r.Body).Decode(&params); err != nil {\n")
	fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusBadRequest)\n")
	fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf(\"invalid request body\")})\n")
	fmt.Fprintf(out, "\t\treturn\n")
	fmt.Fprintf(out, "\t}\n")

	// Валидация
	validationCode := &strings.Builder{}
	generateValidation(validationCode, fields)
	out.WriteString(validationCode.String())

	// Выполнение бизнес-логики
	fmt.Fprintf(out, "\tresult, err := h.%s(context.Background(), params)\n", methodName)
	fmt.Fprintf(out, "\tif err != nil {\n")
	fmt.Fprintf(out, "\t\tw.WriteHeader(http.StatusInternalServerError)\n")
	fmt.Fprintf(out, "\t\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http.StatusInternalServerError, Err: err})\n")
	fmt.Fprintf(out, "\t\treturn\n")
	fmt.Fprintf(out, "\t}\n")

	// Успешный ответ
	fmt.Fprintf(out, "\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
	fmt.Fprintf(out, "\tjson.NewEncoder(w).Encode(result)\n")
	fmt.Fprintf(out, "}\n\n")
}

// generateServeHTTP генерирует реализацию ServeHTTP для структуры
func generateServeHTTP(out *os.File, structName string, specs []ApiSpec) {
	fmt.Fprintf(out, "func (h *%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", structName)
	for _, spec := range specs {
		fmt.Fprintf(out, "\tif r.Method == \"%s\" && r.URL.Path == \"%s\" {\n", spec.Method, spec.URL)
		fmt.Fprintf(out, "\t\th.handler%s(w, r)\n", spec.MethodName)
		fmt.Fprintf(out, "\t\treturn\n")
		fmt.Fprintf(out, "\t}\n") // Закрывающее условие для каждого API специфика
	}
	// Добавлена проверка для всех остальных случаев (404)
	fmt.Fprintf(out, "\tw.WriteHeader(http.StatusNotFound)\n")
	fmt.Fprintln(out, "\tjson.NewEncoder(w).Encode(ApiError{HTTPStatus: http."+
		"StatusNotFound, Err: fmt.Errorf(\"unknown method %s on %s\", r.Method, r.URL.Path)})\n")
	fmt.Fprintf(out, "}\n") // Закрывающая скобка для ServeHTTP
}

// main анализирует исходный код и генерирует серверную обвязку
func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: codegen <input.go> <output.go>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Failed to parse input file: %v", err)
	}

	out, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer out.Close()

	fmt.Fprintf(out, "package %s\n\n", node.Name.Name)
	fmt.Fprintf(out, "import (\n\t\"context\"\n\t\"encoding/json\"\n\t\"net/http\"\n\t\"strings\"\n)\n\n")

	apiHandlers := map[string][]ApiSpec{}

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Doc != nil && strings.HasPrefix(fn.Doc.Text(), "apigen:api") {
				var apiSpec ApiSpec
				err := json.Unmarshal([]byte(strings.TrimPrefix(fn.Doc.Text(), "apigen:api ")), &apiSpec)
				if err != nil {
					log.Fatalf("Failed to parse apigen spec: %v", err)
				}

				methodName := fn.Name.Name
				apiSpec.MethodName = methodName // сохраняем имя метода в структуре
				structName := fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
				paramType := ""

				// Определяем тип параметров из второго параметра метода
				if len(fn.Type.Params.List) > 1 {
					if ident, ok := fn.Type.Params.List[1].Type.(*ast.Ident); ok {
						paramType = ident.Name
					} else {
						log.Fatalf("Unsupported parameter type in method %s", methodName)
					}
				} else {
					log.Fatalf("Method %s does not have parameters", methodName)
				}

				// Находим структуру параметров
				for _, decl := range node.Decls {
					if gen, ok := decl.(*ast.GenDecl); ok {
						for _, spec := range gen.Specs {
							if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == paramType {
								if st, ok := ts.Type.(*ast.StructType); ok {
									fields := parseStructFields(st)
									generateHandler(out, structName, methodName, apiSpec, paramType, fields)

									// Сохраняем для генерации ServeHTTP
									apiHandlers[structName] = append(apiHandlers[structName], apiSpec)
								}
							}
						}
					}
				}
			}
		}
	}

	// Генерируем ServeHTTP для каждой структуры
	for structName, specs := range apiHandlers {
		generateServeHTTP(out, structName, specs)
	}
}
