import { workspace, ExtensionContext, window } from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

/**
 * Starts the goal language server and binds it to .goal documents so the editor
 * shows the language's check violations inline.
 */
export function activate(_context: ExtensionContext): void {
  const cfg = workspace.getConfiguration("goal");
  if (!cfg.get<boolean>("lsp.enable", true)) {
    return;
  }

  const command = cfg.get<string>("lsp.path", "goal");
  const serverOptions: ServerOptions = {
    command,
    args: ["lsp"],
    transport: TransportKind.stdio,
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ language: "goal" }],
  };

  client = new LanguageClient(
    "goal",
    "Goal Language Server",
    serverOptions,
    clientOptions,
  );

  client.start().catch((err) => {
    window.showErrorMessage(
      `Goal language server failed to start (command: "${command} lsp"). ` +
        `Set "goal.lsp.path" to your goal binary. ${err}`,
    );
  });
}

/** Stops the language server when the extension is unloaded. */
export function deactivate(): Thenable<void> | undefined {
  return client?.stop();
}
