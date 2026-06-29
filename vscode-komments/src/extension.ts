import * as vscode from "vscode";
import * as path from "path";
import * as cli from "./cli.js";
import { KommentsTreeProvider, CommentItem } from "./treeView.js";
import { updateDecorations } from "./decorations.js";

let treeProvider: KommentsTreeProvider;

function refreshAll(): void {
  treeProvider.refresh();
  const editor = vscode.window.activeTextEditor;
  if (editor) updateDecorations(editor);
}

function relativeFilePath(uri: vscode.Uri): string {
  const folders = vscode.workspace.workspaceFolders;
  if (folders && folders.length > 0) {
    return path.relative(folders[0].uri.fsPath, uri.fsPath);
  }
  return uri.fsPath;
}

export function activate(context: vscode.ExtensionContext): void {
  treeProvider = new KommentsTreeProvider();

  const treeView = vscode.window.createTreeView("komments.commentsView", {
    treeDataProvider: treeProvider,
    showCollapseAll: false,
  });
  context.subscriptions.push(treeView);

  // Refresh decorations on editor changes
  context.subscriptions.push(
    vscode.window.onDidChangeActiveTextEditor((editor) => {
      if (editor) updateDecorations(editor);
    })
  );

  context.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument(() => {
      const editor = vscode.window.activeTextEditor;
      if (editor) updateDecorations(editor);
    })
  );

  // Initial decoration
  if (vscode.window.activeTextEditor) {
    updateDecorations(vscode.window.activeTextEditor);
  }

  // Commands
  context.subscriptions.push(
    vscode.commands.registerCommand("komments.addComment", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor) return;

      const text = await vscode.window.showInputBox({
        prompt: "Enter your comment",
        placeHolder: "Annotation text...",
      });
      if (!text) return;

      const line = editor.selection.active.line + 1;
      const file = relativeFilePath(editor.document.uri);

      try {
        cli.addComment(file, String(line), text);
        vscode.window.showInformationMessage(
          `Komment added at ${file}:${line}`
        );
        refreshAll();
      } catch (e: any) {
        vscode.window.showErrorMessage(`Komments: ${e.message}`);
      }
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand(
      "komments.addCommentAtSelection",
      async () => {
        const editor = vscode.window.activeTextEditor;
        if (!editor) return;

        const text = await vscode.window.showInputBox({
          prompt: "Enter your comment for the selected range",
          placeHolder: "Annotation text...",
        });
        if (!text) return;

        const startLine = editor.selection.start.line + 1;
        const endLine = editor.selection.end.line + 1;
        const file = relativeFilePath(editor.document.uri);
        const pos = startLine === endLine ? String(startLine) : `${startLine}-${endLine}`;

        try {
          cli.addComment(file, pos, text);
          vscode.window.showInformationMessage(
            `Komment added at ${file}:${pos}`
          );
          refreshAll();
        } catch (e: any) {
          vscode.window.showErrorMessage(`Komments: ${e.message}`);
        }
      }
    )
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("komments.refreshComments", () => {
      refreshAll();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("komments.toggleArchived", () => {
      treeProvider.toggleArchived();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand(
      "komments.archiveComment",
      (item: CommentItem) => {
        try {
          cli.archiveComment(item.comment.id);
          vscode.window.showInformationMessage(
            `Komment #${item.comment.id} archived.`
          );
          refreshAll();
        } catch (e: any) {
          vscode.window.showErrorMessage(`Komments: ${e.message}`);
        }
      }
    )
  );

  context.subscriptions.push(
    vscode.commands.registerCommand(
      "komments.unarchiveComment",
      (item: CommentItem) => {
        try {
          cli.unarchiveComment(item.comment.id);
          vscode.window.showInformationMessage(
            `Komment #${item.comment.id} unarchived.`
          );
          refreshAll();
        } catch (e: any) {
          vscode.window.showErrorMessage(`Komments: ${e.message}`);
        }
      }
    )
  );

  context.subscriptions.push(
    vscode.commands.registerCommand(
      "komments.editComment",
      async (item: CommentItem) => {
        const newText = await vscode.window.showInputBox({
          prompt: `Edit comment #${item.comment.id}`,
          value: item.comment.text,
        });
        if (newText === undefined) return;

        try {
          cli.editComment(item.comment.id, newText);
          vscode.window.showInformationMessage(
            `Komment #${item.comment.id} updated.`
          );
          refreshAll();
        } catch (e: any) {
          vscode.window.showErrorMessage(`Komments: ${e.message}`);
        }
      }
    )
  );

  context.subscriptions.push(
    vscode.commands.registerCommand(
      "komments.deleteComment",
      async (item: CommentItem) => {
        const confirm = await vscode.window.showWarningMessage(
          `Permanently delete comment #${item.comment.id}?`,
          { modal: true },
          "Delete"
        );
        if (confirm !== "Delete") return;

        try {
          cli.deleteComment(item.comment.id);
          vscode.window.showInformationMessage(
            `Komment #${item.comment.id} deleted.`
          );
          refreshAll();
        } catch (e: any) {
          vscode.window.showErrorMessage(`Komments: ${e.message}`);
        }
      }
    )
  );

  context.subscriptions.push(
    vscode.commands.registerCommand(
      "komments.goToComment",
      async (item: CommentItem) => {
        const folders = vscode.workspace.workspaceFolders;
        if (!folders || folders.length === 0) return;

        const filePath = path.join(
          folders[0].uri.fsPath,
          item.comment.file
        );
        const uri = vscode.Uri.file(filePath);

        const line =
          item.comment.position.type === "cursor"
            ? item.comment.position.line - 1
            : item.comment.position.start_line - 1;

        const doc = await vscode.workspace.openTextDocument(uri);
        const editor = await vscode.window.showTextDocument(doc);
        const pos = new vscode.Position(line, 0);
        editor.selection = new vscode.Selection(pos, pos);
        editor.revealRange(
          new vscode.Range(pos, pos),
          vscode.TextEditorRevealType.InCenter
        );
      }
    )
  );
}

export function deactivate(): void {}
