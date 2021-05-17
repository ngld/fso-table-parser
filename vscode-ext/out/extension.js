"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deactivate = exports.activate = void 0;
const vscode_1 = require("vscode");
const node_1 = require("vscode-languageclient/node");
let client;
let output;
function startClient(context) {
    const lspBin = context.asAbsolutePath('server');
    const serverOptions = { command: lspBin };
    const clientOptions = {
        documentSelector: [{ scheme: 'file', language: 'fso-table' }],
        outputChannel: output,
    };
    client = new node_1.LanguageClient('fsoTables', 'FSO Tables LSP', serverOptions, clientOptions, true);
    client.trace = node_1.Trace.Verbose;
    client.start();
}
function activate(context) {
    output = vscode_1.window.createOutputChannel('FSO Tables');
    startClient(context);
    context.subscriptions.push(vscode_1.commands.registerCommand('fso-tables.restart', () => {
        client.stop();
        setTimeout(() => {
            startClient(context);
        }, 500);
    }));
    console.log('FSO Tables LSP sees the light of day!');
}
exports.activate = activate;
function deactivate() {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
exports.deactivate = deactivate;
//# sourceMappingURL=extension.js.map