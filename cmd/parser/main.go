package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ngld/fso-table-parser/pkg/parser"
	"github.com/ngld/fso-table-parser/pkg/structs"
)

func main() {
	ctx := context.Background()
	if len(os.Args) < 2 {
		os.Stderr.WriteString("Usage: parser <path to .tbl or .tbm>\n")
		os.Exit(2)
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Error: Failed to open file: %+v\n", err))
		os.Exit(1)
	}

	lexer := parser.NewLexer(ctx, strings.NewReader(string(content)))
	results := make([]interface{}, 0)
	for _, field := range structs.NewShipsTable() {
		result, err := field.Parse(lexer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				os.Stderr.WriteString(fmt.Sprintf("Failed to parse %s: reached end of file before the section was read or an error ocurred during this section.\n", field.Name))
			} else {
				os.Stderr.WriteString(fmt.Sprintf("Failed to parse %s: %+v\n", field.Name, err))
			}

			for _, err := range lexer.Errors() {
				if !errors.Is(err, io.EOF) {
					os.Stderr.WriteString(fmt.Sprintf("%s\n", err))
				}
			}
			os.Exit(1)
		}

		if result != nil {
			results = append(results, result)
		}
	}

	output, err := json.Marshal(results)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to generate JSON: %+v\n", err))
		os.Exit(1)
	}

	fmt.Print(string(output))
}
