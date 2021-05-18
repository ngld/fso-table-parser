import { window, workspace, commands, ExtensionContext, OutputChannel } from 'vscode';
import * as fs from 'fs-extra';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	Trace
} from 'vscode-languageclient/node';

let client: LanguageClient;
let output: OutputChannel;

async function startClient(context: ExtensionContext): Promise<void> {
	const lspBin = workspace.getConfiguration('fso-tables').get<string | null>('lspBin');
	if (lspBin === null || lspBin === undefined) return;

	try {
		await fs.stat(lspBin);
	} catch (e) {
		if (e?.code === 'ENOENT') {
			window.showWarningMessage('The LSP binary could not be found.');
		} else {
			window.showErrorMessage(`Could not access the LSP binary: ${(e as Error).message}`);
		}
		return;
	}

	const serverOptions: ServerOptions = { command: lspBin };
	const clientOptions: LanguageClientOptions = {
		documentSelector: [{ scheme: 'file', language: 'fso-table' }],
		outputChannel: output,
		traceOutputChannel: output,
	};

	client = new LanguageClient('fsoTables', 'FSO Tables LSP', serverOptions, clientOptions, true);
	client.trace = Trace.Verbose;
	client.clientOptions.errorHandler = client.createDefaultErrorHandler(4);

	context.subscriptions.push(client.start());
	output.appendLine('Launched LSP');
}

export function activate(context: ExtensionContext) {
	output = window.createOutputChannel('FSO Tables');
	context.subscriptions.push(output);

	startClient(context);

	context.subscriptions.push(commands.registerCommand('fso-tables.restart', () => {
		output.appendLine('Restarting LSP');
		if (client) client.stop();

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
