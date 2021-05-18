package lsp

import (
	contextpkg "context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	uri       string
	content   string
	ctx       contextpkg.Context
	ctxCancel contextpkg.CancelFunc
	scopes    []parser.ScopeInfo
	sync.Mutex
	version        int32
	pendingVersion int32
}

func processLexerErrors(errors []error, severity protocol.DiagnosticSeverity, content string) []protocol.Diagnostic {
	msgs := make([]protocol.Diagnostic, len(errors))
	for idx, err := range errors {
		var loc [4]int
		if errInfo, ok := eris.Cause(err).(parser.ParserError); ok {
			loc = errInfo.Location()
		} else {
			loc = [4]int{0, 0, 0, 0}
		}
		msgs[idx] = protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(loc[0] - 1),
					Character: uint32(loc[1]),
				},
				End: protocol.Position{
					Line:      uint32(loc[2] - 1),
					Character: uint32(loc[3]),
				},
			},
			Severity: &severity,
			Code: &protocol.IntegerOrString{
				Value: "fso-lsp-error",
			},
			Message: err.Error(),
		}
	}

	return msgs
}

func analyseDoc(context *glsp.Context, doc *docCacheEntry, fields []parser.ContainerItem) {
	defer func() {
		p := recover()
		if p != nil {
			err := eris.New(fmt.Sprint(p))
			protocol.Trace(context, protocol.MessageTypeError, eris.ToString(err, true))
		}
	}()

	doc.ctx, doc.ctxCancel = contextpkg.WithDeadline(contextpkg.Background(), time.Now().Add(1000*time.Millisecond))

	protocol.Trace(context, protocol.MessageTypeInfo, fmt.Sprintf("Parsing %s", doc.uri))
	start := time.Now()
	lexer := parser.NewLexer(doc.ctx, strings.NewReader(doc.content))

	for _, container := range fields {
		if doc.ctx.Err() != nil {
			break
		}
		_, err := container.Parse(lexer)
		if err != nil {
			lexer.Report(err)
		}
	}

	if doc.ctx.Err() != nil {
		protocol.Trace(context, protocol.MessageTypeInfo, fmt.Sprintf("Canceled %s (%v)", doc.uri, doc.ctx.Err()))
		return
	}

	msgs := processLexerErrors(lexer.Errors(), protocol.DiagnosticSeverityError, doc.content)
	msgs = append(msgs, processLexerErrors(lexer.Warnings(), protocol.DiagnosticSeverityInformation, doc.content)...)
	end := time.Now()

	version := uint32(doc.version)
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         doc.uri,
		Version:     &version,
		Diagnostics: msgs,
	})

	doc.scopes = lexer.ScopeInfos()

	duration := end.Sub(start).Milliseconds()
	protocol.Trace(context, protocol.MessageTypeInfo, fmt.Sprintf("Processed %s in %dms", doc.uri, duration))
	doc.ctxCancel()
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
			/*if params.Trace != nil {
				protocol.SetTraceValue(*params.Trace)
			}*/
			protocol.SetTraceValue(protocol.TraceValueVerbose)
			protocol.Trace(context, protocol.MessageTypeInfo, fmt.Sprintf("Trace: %v", *params.Trace))

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
		Shutdown: func(context *glsp.Context) error {
			return nil
		},
		Exit: func(context *glsp.Context) error {
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
			atomic.StoreInt32(&item.pendingVersion, params.TextDocument.Version)
			item.ctxCancel()

			go func() {
				// Run the text updates in a goroutine to avoid blocking the server on the lock
				item.Lock()
				item.version = params.TextDocument.Version

				for _, change := range params.ContentChanges {
					if ev, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
						start, end := ev.Range.IndexesIn(item.content)
						item.content = item.content[:start] + ev.Text + item.content[end:]
					} else if ev, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
						item.content = ev.Text
					}
				}

				if item.pendingVersion == params.TextDocument.Version {
					// Only trigger the analysis for the latest version
					analyseDoc(context, item, fields)
				}
				item.Unlock()
			}()

			return nil
		},
		TextDocumentDidClose: func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
			delete(docCache, params.TextDocument.URI)
			return nil
		},

		TextDocumentHover: func(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
			doc, found := docCache[params.TextDocument.URI]
			if !found {
				return nil, eris.Errorf("Document %s not found", params.TextDocument.URI)
			}

			line := int(params.Position.Line) + 1
			col := int(params.Position.Character)
			for _, info := range doc.scopes {
				if info.Start[0] <= line && info.Start[1] <= col &&
					info.End[0] >= line && info.End[1] >= col {
					return &protocol.Hover{
						Range: &protocol.Range{
							Start: protocol.Position{
								Line:      uint32(info.Start[0]) - 1,
								Character: uint32(info.Start[1]),
							},
							End: protocol.Position{
								Line:      uint32(info.End[0]) - 1,
								Character: uint32(info.End[1]),
							},
						},
						Contents: protocol.MarkupContent{
							Kind:  protocol.MarkupKindPlainText,
							Value: info.HoverText,
						},
					}, nil
				}
			}
			return nil, nil
		},
	}

	return handler
}
