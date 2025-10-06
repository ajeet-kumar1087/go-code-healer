package ai

import (
	"go/parser"
	"go/token"
	"strings"
)

// CodeValidator handles validation of generated code
type CodeValidator struct {
	logger Logger
}

// NewCodeValidator creates a new code validator
func NewCodeValidator(logger Logger) *CodeValidator {
	return &CodeValidator{
		logger: logger,
	}
}

// ValidateGoSyntax performs basic Go syntax validation on the proposed fix
func (cv *CodeValidator) ValidateGoSyntax(code string) bool {
	if code == "" {
		return false
	}

	// Create a file set for parsing
	fset := token.NewFileSet()

	// Try to parse as a complete Go file first
	if _, err := parser.ParseFile(fset, "", "package main\n"+code, parser.ParseComments); err == nil {
		return true
	}

	// Try to parse as a function
	funcCode := "package main\nfunc dummy() {\n" + code + "\n}"
	if _, err := parser.ParseFile(fset, "", funcCode, parser.ParseComments); err == nil {
		return true
	}

	// Try to parse as expressions/statements
	stmtCode := "package main\nfunc dummy() {\n" + code + "\n}"
	if _, err := parser.ParseFile(fset, "", stmtCode, parser.ParseComments); err == nil {
		return true
	}

	// Try to parse as a declaration
	declCode := "package main\n" + code
	if _, err := parser.ParseFile(fset, "", declCode, parser.ParseComments); err == nil {
		return true
	}

	// If all parsing attempts fail, check for basic Go syntax elements
	return cv.basicSyntaxCheck(code)
}

// basicSyntaxCheck performs basic syntax validation without full parsing
func (cv *CodeValidator) basicSyntaxCheck(code string) bool {
	// Check for balanced braces, brackets, and parentheses
	braces := 0
	brackets := 0
	parens := 0
	inString := false
	inChar := false
	escaped := false

	for i, r := range code {
		if escaped {
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if inString {
			if r == '"' {
				inString = false
			}
			continue
		}

		if inChar {
			if r == '\'' {
				inChar = false
			}
			continue
		}

		switch r {
		case '"':
			inString = true
		case '\'':
			inChar = true
		case '{':
			braces++
		case '}':
			braces--
		case '[':
			brackets++
		case ']':
			brackets--
		case '(':
			parens++
		case ')':
			parens--
		}

		// Early exit if we have negative counts (unbalanced)
		if braces < 0 || brackets < 0 || parens < 0 {
			if cv.logger != nil {
				cv.logger.Debug("Syntax validation failed: unbalanced delimiters at position %d", i)
			}
			return false
		}
	}

	// Check if all delimiters are balanced
	balanced := braces == 0 && brackets == 0 && parens == 0

	if !balanced && cv.logger != nil {
		cv.logger.Debug("Syntax validation failed: unbalanced delimiters (braces: %d, brackets: %d, parens: %d)",
			braces, brackets, parens)
	}

	return balanced
}

// AssessErrorComplexity analyzes the error to determine its complexity level
func (cv *CodeValidator) AssessErrorComplexity(request FixRequest) string {
	errorLower := strings.ToLower(request.Error)
	stackLower := strings.ToLower(request.StackTrace)

	// Simple, common errors
	simplePatterns := []string{
		"nil pointer dereference",
		"index out of range",
		"slice bounds out of range",
		"invalid memory address",
		"assignment to entry in nil map",
	}

	for _, pattern := range simplePatterns {
		if strings.Contains(errorLower, pattern) {
			return "simple"
		}
	}

	// Complex errors
	complexPatterns := []string{
		"deadlock",
		"race condition",
		"goroutine",
		"channel",
		"interface conversion",
		"reflection",
		"unsafe",
	}

	for _, pattern := range complexPatterns {
		if strings.Contains(errorLower, pattern) || strings.Contains(stackLower, pattern) {
			return "complex"
		}
	}

	// Check stack trace depth - deeper stacks might indicate more complex issues
	stackLines := strings.Count(request.StackTrace, "\n")
	if stackLines > 20 {
		return "complex"
	} else if stackLines < 5 {
		return "simple"
	}

	return "moderate"
}
