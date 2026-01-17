package service

import (
	"strings"

	"github.com/go-enry/go-enry/v2"
)

// languageToSyntax maps enry language names to our syntax type names
var languageToSyntax = map[string]string{
	"Python":       "python",
	"JavaScript":   "javascript",
	"TypeScript":   "typescript",
	"Go":           "go",
	"Java":         "java",
	"C":            "c",
	"C++":          "cpp",
	"C#":           "csharp",
	"Ruby":         "ruby",
	"PHP":          "php",
	"Rust":         "rust",
	"Swift":        "swift",
	"Kotlin":       "kotlin",
	"Scala":        "scala",
	"SQL":          "sql",
	"Shell":        "bash",
	"Bash":         "bash",
	"PowerShell":   "powershell",
	"YAML":         "yaml",
	"TOML":         "toml",
	"INI":          "ini",
	"Dockerfile":   "dockerfile",
	"Makefile":     "makefile",
	"Lua":          "lua",
	"Perl":         "perl",
	"R":            "r",
	"MATLAB":       "matlab",
	"TeX":          "latex",
	"LaTeX":        "latex",
	"Diff":         "diff",
	"GraphQL":      "graphql",
	"Protocol Buffer": "protobuf",
	"Haskell":      "haskell",
	"Elixir":       "elixir",
	"Erlang":       "erlang",
	"Clojure":      "clojure",
	"Common Lisp":  "lisp",
	"Emacs Lisp":   "lisp",
	"Scheme":       "lisp",
	"Vim Script":   "vim",
	"Vim script":   "vim",
	"VimL":         "vim",
	"Assembly":     "assembly",
	"HTML":         "html",
	"CSS":          "css",
	"JSON":         "json",
	"XML":          "xml",
	"Markdown":     "markdown",
	"Nginx":        "nginx",
	"Apache":       "apache",
	"Text":         "plaintext",
}

// SyntaxDetector provides language detection functionality
type SyntaxDetector struct{}

// NewSyntaxDetector creates a new SyntaxDetector instance
func NewSyntaxDetector() *SyntaxDetector {
	return &SyntaxDetector{}
}

// DetectLanguage attempts to detect the programming language from content
// Returns the detected syntax type or "plaintext" if detection fails
func (d *SyntaxDetector) DetectLanguage(content string) string {
	if content == "" {
		return DefaultSyntaxType
	}

	// Use enry to detect language from content
	language := enry.GetLanguage("", []byte(content))

	if language == "" {
		// Try to detect by looking at common patterns
		return d.detectByPatterns(content)
	}

	// Map enry language to our syntax type
	if syntax, ok := languageToSyntax[language]; ok {
		return syntax
	}

	// If no mapping, try lowercase of language name
	lowercaseLang := strings.ToLower(language)
	if ValidSyntaxTypes[lowercaseLang] {
		return lowercaseLang
	}

	return DefaultSyntaxType
}

// detectByPatterns attempts to detect language using common patterns
func (d *SyntaxDetector) detectByPatterns(content string) string {
	content = strings.TrimSpace(content)

	// Check for shebang
	if strings.HasPrefix(content, "#!") {
		firstLine := strings.Split(content, "\n")[0]
		switch {
		case strings.Contains(firstLine, "python"):
			return "python"
		case strings.Contains(firstLine, "bash") || strings.Contains(firstLine, "/sh"):
			return "bash"
		case strings.Contains(firstLine, "node"):
			return "javascript"
		case strings.Contains(firstLine, "ruby"):
			return "ruby"
		case strings.Contains(firstLine, "perl"):
			return "perl"
		case strings.Contains(firstLine, "php"):
			return "php"
		}
	}

	// Check for JSON
	if (strings.HasPrefix(content, "{") && strings.HasSuffix(strings.TrimSpace(content), "}")) ||
		(strings.HasPrefix(content, "[") && strings.HasSuffix(strings.TrimSpace(content), "]")) {
		return "json"
	}

	// Check for XML/HTML
	if strings.HasPrefix(content, "<?xml") {
		return "xml"
	}
	if strings.HasPrefix(content, "<!DOCTYPE") || strings.HasPrefix(content, "<html") {
		return "html"
	}

	// Check for YAML
	if strings.Contains(content, "---\n") || (strings.Contains(content, ":") && !strings.Contains(content, ";")) {
		lines := strings.Split(content, "\n")
		yamlLike := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, ":") && !strings.HasSuffix(line, ";") {
				yamlLike++
			}
		}
		if yamlLike > 2 {
			return "yaml"
		}
	}

	// Check for common language keywords
	if strings.Contains(content, "def ") && strings.Contains(content, ":") {
		return "python"
	}
	if strings.Contains(content, "func ") && strings.Contains(content, "package ") {
		return "go"
	}
	if strings.Contains(content, "function ") || strings.Contains(content, "const ") || strings.Contains(content, "let ") {
		return "javascript"
	}
	if strings.Contains(content, "public class ") || strings.Contains(content, "private ") {
		return "java"
	}

	return DefaultSyntaxType
}

// DetectLanguageWithFilename attempts to detect language using both filename and content
// Filename takes precedence if it provides a clear match
func (d *SyntaxDetector) DetectLanguageWithFilename(filename, content string) string {
	if filename != "" {
		// Use enry to detect by filename first
		language := enry.GetLanguage(filename, []byte(content))
		if language != "" {
			if syntax, ok := languageToSyntax[language]; ok {
				return syntax
			}
			lowercaseLang := strings.ToLower(language)
			if ValidSyntaxTypes[lowercaseLang] {
				return lowercaseLang
			}
		}
	}

	// Fallback to content-only detection
	return d.DetectLanguage(content)
}
