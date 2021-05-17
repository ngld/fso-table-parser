package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/ngld/fso-table-parser/pkg/parser"
	"github.com/ngld/fso-table-parser/pkg/structs"
	"github.com/rotisserie/eris"
)

func main() {
	data, err := ioutil.ReadFile("test.tbl")
	if err != nil {
		panic(err)
	}

	defer func() {
		r := recover()
		if r != nil {
			err := eris.New(fmt.Sprint(r))
			panic(eris.ToString(err, true))
		}
	}()

	lexer := parser.NewLexer(strings.NewReader(string(data)))
	containers := structs.NewTestTable()
	for _, container := range containers {
		result, err := container.Parse(lexer)
		if err != nil {
			fmt.Println(eris.ToString(err, true))
			os.Exit(1)
		}

		spew.Dump(result)
	}
}
