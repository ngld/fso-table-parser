package lsp

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ngld/fso-table-parser/pkg/parser"
	"github.com/ngld/fso-table-parser/pkg/structs"
	"github.com/rotisserie/eris"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	version = "0.0.1"
	lspCode = "fso-lsp-error"
)

type docCacheEntry struct {
	uri     string
	content string
	tree    []interface{}
	errors  []protocol.Diagnostic
	sync.Mutex
	version int32
}

func analyseDoc(context *glsp.Context, doc *docCacheEntry, fields []parser.ContainerItem) {
	defer func() {
		p := recover()
		if p != nil {
			err := eris.New(fmt.Sprint(p))
			protocol.Trace(context, protocol.MessageTypeError, eris.ToString(err, true))
		}
	}()

	doc.Lock()
	defer doc.Unlock()

	lexer := parser.NewLexer(strings.NewReader(doc.content))
	doc.errors = make([]protocol.Diagnostic, 0)
	doc.tree = make([]interface{}, len(fields))

	for idx, container := range fields {
		val, err := container.Parse(lexer)
		if err != nil {
			var loc [2]int
			if errInfo, ok := eris.Cause(err).(parser.ParserError); ok {
				loc = errInfo.Location()
			} else {
				loc = [2]int{0, 0}
			}
			position := protocol.Position{
				Line:      uint32(loc[0] - 1),
				Character: uint32(loc[1]),
			}
			severity := protocol.DiagnosticSeverityError
			doc.errors = append(doc.errors, protocol.Diagnostic{
				Range: protocol.Range{
					Start: position,
					End:   position.EndOfLineIn(doc.content),
				},
				Severity: &severity,
				Code: &protocol.IntegerOrString{
					Value: "fso-lsp-error",
				},
				Message: err.Error(),
			})
		} else {
			doc.tree[idx] = val
		}
	}

	version := uint32(doc.version)
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         doc.uri,
		Version:     &version,
		Diagnostics: doc.errors,
	})
}

func GetHandler() *protocol.Handler {
	var handler *protocol.Handler
	docCache := make(map[string]*docCacheEntry)
	fields := structs.NewTestTable()

	handler = &protocol.Handler{
		CancelRequest: func(context *glsp.Context, params *protocol.CancelParams) error {
			return nil
		},
		Progress: func(context *glsp.Context, params *protocol.ProgressParams) error {
			return nil
		},

		Initialize: func(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
			if params.Trace != nil {
				protocol.SetTraceValue(*params.Trace)
			}

			caps := handler.CreateServerCapabilities()
			caps.TextDocumentSync = protocol.TextDocumentSyncKindIncremental

			protocol.Trace(context, protocol.MessageTypeInfo, "Hello World!")
			return protocol.InitializeResult{
				Capabilities: caps,
				ServerInfo: &protocol.InitializeResultServerInfo{
					Name:    "FSO Tables LSP",
					Version: &version,
				},
			}, nil
		},
		Initialized: func(context *glsp.Context, params *protocol.InitializedParams) error {
			return nil
		},

		LogTrace: func(context *glsp.Context, params *protocol.LogTraceParams) error {
			return nil
		},
		SetTrace: func(context *glsp.Context, params *protocol.SetTraceParams) error {
			protocol.SetTraceValue(params.Value)
			return nil
		},

		TextDocumentDidOpen: func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
			doc := params.TextDocument
			docCache[doc.URI] = &docCacheEntry{
				uri:     doc.URI,
				version: doc.Version,
				content: doc.Text,
				tree:    nil,
			}

			go analyseDoc(context, docCache[doc.URI], fields)
			return nil
		},
		TextDocumentDidChange: func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
			defer func() {
				p := recover()
				if p != nil {
					err := eris.New(fmt.Sprint(p))
					protocol.Trace(context, protocol.MessageTypeError, eris.ToString(err, true))
				}
			}()

			item := docCache[params.TextDocument.URI]
			item.Lock()
			defer item.Unlock()

			item.version = params.TextDocument.Version
			item.tree = nil

			for _, change := range params.ContentChanges {
				if ev, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
					start, end := ev.Range.IndexesIn(item.content)
					item.content = item.content[:start] + ev.Text + item.content[end:]
				} else if ev, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
					item.content = ev.Text
				}
			}

			go analyseDoc(context, item, fields)
			return nil
		},
		TextDocumentDidClose: func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
			delete(docCache, params.TextDocument.URI)
			return nil
		},
	}

	return handler
}
