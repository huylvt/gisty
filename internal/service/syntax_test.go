package service

import (
	"testing"
)

func TestSyntaxDetector_DetectLanguage(t *testing.T) {
	detector := NewSyntaxDetector()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: "plaintext",
		},
		{
			name: "python function",
			content: `def hello():
    print("Hello, World!")

if __name__ == "__main__":
    hello()`,
			expected: "python",
		},
		{
			name: "go code",
			content: `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`,
			expected: "go",
		},
		{
			name: "javascript code",
			content: `function hello() {
    console.log("Hello, World!");
}

const greeting = "Hello";
let name = "World";`,
			expected: "javascript",
		},
		{
			name: "json object",
			content: `{
    "name": "John",
    "age": 30,
    "city": "New York"
}`,
			expected: "json",
		},
		{
			name:     "json array",
			content:  `[1, 2, 3, 4, 5]`,
			expected: "json",
		},
		{
			name: "html document",
			content: `<!DOCTYPE html>
<html>
<head>
    <title>Test</title>
</head>
<body>
    <h1>Hello, World!</h1>
</body>
</html>`,
			expected: "html",
		},
		{
			name: "xml document",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<root>
    <item>Hello</item>
</root>`,
			expected: "xml",
		},
		{
			name: "bash script with shebang",
			content: `#!/bin/bash
echo "Hello, World!"
for i in 1 2 3; do
    echo $i
done`,
			expected: "bash",
		},
		{
			name: "python script with shebang",
			content: `#!/usr/bin/env python3
print("Hello, World!")`,
			expected: "python",
		},
		{
			name: "yaml config",
			content: `name: my-app
version: 1.0.0
dependencies:
  - express
  - lodash
config:
  port: 3000
  debug: true`,
			expected: "yaml",
		},
		{
			name: "java class",
			content: `public class HelloWorld {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }
}`,
			expected: "java",
		},
		{
			name:     "plain text",
			content:  "This is just some plain text without any code.",
			expected: "plaintext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectLanguage(tt.content)
			if result != tt.expected {
				t.Errorf("DetectLanguage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSyntaxDetector_DetectLanguageWithFilename(t *testing.T) {
	detector := NewSyntaxDetector()

	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			name:     "python file",
			filename: "script.py",
			content:  "x = 1",
			expected: "python",
		},
		{
			name:     "go file",
			filename: "main.go",
			content:  "package main",
			expected: "go",
		},
		{
			name:     "javascript file",
			filename: "app.js",
			content:  "const x = 1;",
			expected: "javascript",
		},
		{
			name:     "typescript file",
			filename: "app.ts",
			content:  "const x: number = 1;",
			expected: "typescript",
		},
		{
			name:     "dockerfile",
			filename: "Dockerfile",
			content:  "FROM ubuntu:latest",
			expected: "dockerfile",
		},
		{
			name:     "makefile",
			filename: "Makefile",
			content:  "all: build",
			expected: "makefile",
		},
		{
			name:     "yaml file",
			filename: "config.yaml",
			content:  "key: value",
			expected: "yaml",
		},
		{
			name:     "json file",
			filename: "package.json",
			content:  `{"name": "test"}`,
			expected: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectLanguageWithFilename(tt.filename, tt.content)
			if result != tt.expected {
				t.Errorf("DetectLanguageWithFilename() = %v, want %v", result, tt.expected)
			}
		})
	}
}
