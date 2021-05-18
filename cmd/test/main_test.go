package main

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ngld/fso-table-parser/pkg/parser"
	"github.com/ngld/fso-table-parser/pkg/structs"
)

func BenchmarkParsingSpeed(b *testing.B) {
	data, err := ioutil.ReadFile("test.tbl")
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := parser.NewLexer(context.Background(), strings.NewReader(string(data)))
		containers := structs.NewTestTable()
		for _, container := range containers {
			container.Parse(lexer)
		}
	}
}
