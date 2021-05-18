package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/ngld/fso-table-parser/pkg/parser"
	"github.com/ngld/fso-table-parser/pkg/structs"
	"github.com/rotisserie/eris"
)

func main() {
	data, err := ioutil.ReadFile("test.tbl")
	if err != nil {
		panic(err)
	}

	f, err := os.Create("profile.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)

	defer func() {
		r := recover()
		if r != nil {
			err := eris.New(fmt.Sprint(r))
			panic(eris.ToString(err, true))
		}
	}()

	lexer := parser.NewLexer(context.Background(), strings.NewReader(string(data)))
	containers := structs.NewTestTable()
	for _, container := range containers {
		_, err := container.Parse(lexer)
		if err != nil {
			fmt.Println(eris.ToString(err, true))
			// os.Exit(1)
		}

		// spew.Dump(result)
	}

	pprof.StopCPUProfile()

	for _, err := range lexer.Errors() {
		fmt.Println(eris.ToString(err, true))
	}
}
