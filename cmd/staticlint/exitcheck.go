// Package main provides a multichecker tool for static analysis of Go code.
//
// This tool aggregates several groups of analyzers:
//
//  1. **Standard Analyzers** from the package
//     "golang.org/x/tools/go/analysis/passes". These check for common mistakes in Go code. For example:
//     - **asmdecl**: Verifies that assembly function declarations match their Go prototypes.
//     - **assign**: Analyzes assignment operators.
//     - **atomic**: Checks correct usage of atomic operations.
//     - **atomicalign**: Ensures proper alignment of atomic values.
//     - **bools**: Analyzes boolean expressions for potential errors.
//     - **buildssa**: Validates the construction of the SSA (Static Single Assignment) form.
//     - **buildtag**: Checks the correctness of build tags.
//     - **cgocall**: Analyzes calls to Cgo.
//     - **composite**: Checks composite literals for correctness.
//     - **copylock**: Detects copying of values that contain locks.
//     - **ctrlflow**: Analyzes control flow in the code.
//     - **deepequalerrors**: Warns against comparing errors using reflect.DeepEqual.
//     - **directive**: Checks the correctness of source directives.
//     - **errorsas**: Analyzes the use of errors.As for error handling.
//     - **fieldalignment**: Examines struct field alignment for optimal memory usage.
//     - **findcall**: Searches for function calls in the code.
//     - **framepointer**: Reports assembly code that clobbers the frame pointer before saving it.
//     - **httpmux** and **httpresponse**: Check correct use of HTTP muxers and responses.
//     - **ifaceassert**: Analyzes type assertions on interface values.
//     - **inspect**: Provides a generic AST traversal mechanism for other analyzers.
//     - **loopclosure**: Detects closures defined within loops that might behave unexpectedly.
//     - **lostcancel**: Checks that cancellation functions (cancel) are not lost.
//     - **nilfunc** and **nilness**: Verify proper handling of nil values.
//     - **pkgfact**: Analyzes package fact data.
//     - **printf**: Checks format strings in formatted output functions.
//     - **reflectvaluecompare**: Warns against questionable comparisons of reflect.Value.
//     - **shadow**: Detects variable shadowing.
//     - **shift**: Verifies correctness of bit-shift operations.
//     - **sigchanyzer**: Analyzes the use of signal channels.
//     - **slog**: Inspects logging function calls.
//     - **sortslice**: Checks the correctness of slice sorting.
//     - **stdmethods**: Analyzes the implementation of standard methods.
//     - **stdversion**: Checks compatibility with the standard library version.
//     - **stringintconv**: Verifies conversions between strings and numeric types.
//     - **structtag**: Validates struct field tags.
//     - **testinggoroutine**: Analyzes the usage of goroutines in tests.
//     - **tests**: Inspects test functions for common mistakes.
//     - **timeformat**: Analyzes time format strings.
//     - **unmarshal**: Checks functions that unmarshal data.
//     - **unreachable**: Detects unreachable code.
//     - **unsafeptr**: Analyzes unsafe pointer usage.
//     - **unusedresult** and **unusedwrite**: Ensure that results of function calls or writes are not unused.
//     - **usesgenerics**: Checks proper usage of generics.
//     - **appends**: Analyzes calls to the built-in append function.
//
//  2. **Staticcheck Analyzers (SA class)** from the package
//     "honnef.co/go/tools/staticcheck". These analyzers detect serious issues in the code.
//     In the multichecker, all analyzers whose names start with "SA" are automatically added.
//     Their settings are defined in a configuration file (e.g., staticcheck.conf with `checks = ["SA*"]`).
//
// 3. **Analyzer ST1005** is added separately (this was previously classified as ST1005).
//
// 4. **Additional Public Analyzers** from external repositories:
//
//   - **forcetypeassert**: Checks the correct usage of type assertions.
//
//   - **wraperrfmt**: Verifies the formatting of wrapped errors.
//
//     5. **Custom Analyzer: ExitCheckAnalyzer**.
//     This analyzer prohibits the direct call to os.Exit in the main function of packages named "main".
//     It works as follows:
//
//   - Analysis is performed only for packages whose import path starts with "github.com/jayjaytrn/URLShortener".
//
//   - Only packages with the name "main" are analyzed.
//
//   - Files located in the vendor directory or marked as generated (with "Code generated" comments)
//     are ignored.
//
//   - If os.Exit is found in the main function, the analyzer reports an error with the file name and line number.
//
// **Multichecker Launch Mechanism:**
// The main function collects all analyzers into a slice and passes them to multichecker.Main.
// When run (for example, via the command:
//
//	go run -trimpath ./cmd/staticlint ./...
//
// the multichecker traverses all packages specified in the arguments and executes each analyzer on the source code.
// The output is a collection of error or warning messages reported by the analyzers.
//
// **Usage:**
// 1. Ensure that a staticcheck.conf file (e.g., with `checks = ["SA*"]`) is located in the project root (next to go.mod).
// 2. Run the command:
//
//	go run -trimpath ./cmd/staticlint ./...
//
// This will analyze the entire codebase.
// For help on a specific analyzer, run:
//
//	./staticlint help <analyzer>
//
// where `<analyzer>` is the name of the analyzer.
package main

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis/passes/appends"
	// Стандартные анализаторы из пакета golang.org/x/tools/go/analysis/passes
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"

	"honnef.co/go/tools/staticcheck"

	"github.com/gostaticanalysis/forcetypeassert"
	"github.com/gostaticanalysis/wraperrfmt"
)

var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "запрещает использование os.Exit в функции main пакета main",
	Run:  runExitCheck,
}

func isGenerated(file *ast.File) bool {
	for _, cg := range file.Comments {
		if strings.Contains(cg.Text(), "Code generated") {
			return true
		}
	}
	return false
}

func runExitCheck(pass *analysis.Pass) (interface{}, error) {
	// Выполняем анализ только для пакетов вашего модуля.
	if !strings.HasPrefix(pass.Pkg.Path(), "github.com/jayjaytrn/URLShortener") {
		return nil, nil
	}
	// Анализируем только пакеты с именем "main".
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	// Проходим по всем файлам пакета.
	for _, file := range pass.Files {
		// Пропускаем сгенерированные файлы (например, сгенерированные go test).
		if isGenerated(file) {
			continue
		}
		// Ищем функцию main.
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				continue
			}
			// Обходим тело функции main в поисках вызова os.Exit.
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if ident.Name == "os" && sel.Sel.Name == "Exit" {
					pos := pass.Fset.Position(call.Pos())
					// Можно также добавить проверку на vendor, если нужно.
					if strings.Contains(pos.Filename, string(filepath.Separator)+"vendor"+string(filepath.Separator)) {
						return true
					}
					pass.Reportf(call.Pos(), "прямой вызов os.Exit в функции main запрещён (файл: %s, строка: %d)", pos.Filename, pos.Line)
				}
				return true
			})
		}
	}
	return nil, nil
}

func main() {
	// Собираем список анализаторов
	analyzers := []*analysis.Analyzer{
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildssa.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		deepequalerrors.Analyzer,
		errorsas.Analyzer,
		directive.Analyzer,
		findcall.Analyzer,
		framepointer.Analyzer,
		httpmux.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		pkgfact.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
		appends.Analyzer,
		// Добавляем собственный анализатор
		ExitCheckAnalyzer,
	}

	// Добавляем все анализаторы класса SA из staticcheck.
	// Фильтруем по префиксу "SA", остальные настройки определяются в staticcheck.conf.
	for _, a := range staticcheck.Analyzers {
		if len(a.Analyzer.Name) >= 2 && a.Analyzer.Name[:2] == "SA" {
			analyzers = append(analyzers, a.Analyzer)
		}
	}

	for _, a := range staticcheck.Analyzers {
		if a.Analyzer.Name == "ST1005" {
			analyzers = append(analyzers, a.Analyzer)
			break
		}
	}

	analyzers = append(analyzers, wraperrfmt.Analyzer, forcetypeassert.Analyzer)

	// Запускаем multichecker со всеми анализаторами.
	multichecker.Main(analyzers...)
}
