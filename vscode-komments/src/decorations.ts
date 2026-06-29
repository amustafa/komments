import * as vscode from "vscode";
import * as path from "path";
import * as cli from "./cli.js";
import type { Comment } from "./types.js";

const decorationType = vscode.window.createTextEditorDecorationType({
  gutterIconPath: undefined,
  after: {
    contentText: " 💬",
    color: new vscode.ThemeColor("editorCodeLens.foreground"),
    fontStyle: "normal",
  },
  isWholeLine: true,
});

export function updateDecorations(editor: vscode.TextEditor): void {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) {
    editor.setDecorations(decorationType, []);
    return;
  }

  const workspaceRoot = folders[0].uri.fsPath;
  const filePath = editor.document.uri.fsPath;
  const relPath = path.relative(workspaceRoot, filePath);

  if (relPath.startsWith("..")) {
    editor.setDecorations(decorationType, []);
    return;
  }

  let comments: Comment[];
  try {
    comments = cli.listComments(false);
  } catch {
    editor.setDecorations(decorationType, []);
    return;
  }

  const fileComments = comments.filter(
    (c) => c.file === relPath || c.file === relPath.replace(/\\/g, "/")
  );

  const decorations: vscode.DecorationOptions[] = fileComments.map((c) => {
    const line =
      c.position.type === "cursor"
        ? c.position.line - 1
        : c.position.start_line - 1;

    const endLine =
      c.position.type === "range"
        ? c.position.end_line - 1
        : line;

    const preview =
      c.text.length > 100
        ? c.text.slice(0, 97) + "..."
        : c.text;

    return {
      range: new vscode.Range(line, 0, endLine, 0),
      hoverMessage: new vscode.MarkdownString(
        `**Komment #${c.id}**\n\n${preview}`
      ),
    };
  });

  editor.setDecorations(decorationType, decorations);
}

export function clearDecorations(editor: vscode.TextEditor): void {
  editor.setDecorations(decorationType, []);
}
