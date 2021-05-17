package main

import (
	"github.com/ngld/fso-table-parser/pkg/lsp"
	"github.com/tliron/glsp/server"
	"github.com/tliron/kutil/logging"
	_ "github.com/tliron/kutil/logging/simple"
)

func main() {
	server := server.NewServer(lsp.GetHandler(), "FSOTBL", true)
	server.Log = logging.GetLogger("LSP")

	err := server.RunStdio()
	if err != nil {
		panic(err)
	}
}
