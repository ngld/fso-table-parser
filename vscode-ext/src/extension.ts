import { window, workspace, commands, ExtensionContext, OutputChannel } from 'vscode';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	Trace
} from 'vscode-languageclient/node';

let client: LanguageClient;
let output: OutputChannel;

function startClient(context: ExtensionContext): void {
	const lspBin = context.asAbsolutePath('server');
	const serverOptions: ServerOptions = { command: lspBin };
	const clientOptions: LanguageClientOptions = {
		documentSelector: [{ scheme: 'file', language: 'fso-table' }],
		outputChannel: output,
	};

	client = new LanguageClient('fsoTables', 'FSO Tables LSP', serverOptions, clientOptions, true);
	client.trace = Trace.Verbose;
	client.start();
}

export function activate(context: ExtensionContext) {
	output = window.createOutputChannel('FSO Tables');
	startClient(context);

	context.subscriptions.push(commands.registerCommand('fso-tables.restart', () => {
		client.stop();
		setTimeout(() => {
			startClient(context);
		}, 500);
	}));

	console.log('FSO Tables LSP sees the light of day!');
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}

	return client.stop();
}
