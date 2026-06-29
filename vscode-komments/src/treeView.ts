import * as vscode from "vscode";
import * as cli from "./cli.js";
import type { Comment } from "./types.js";

export class CommentItem extends vscode.TreeItem {
  constructor(public readonly comment: Comment) {
    const pos =
      comment.position.type === "cursor"
        ? `L${comment.position.line}`
        : `L${comment.position.start_line}-${comment.position.end_line}`;

    const preview =
      comment.text.length > 60
        ? comment.text.slice(0, 57).replace(/\n/g, " ") + "..."
        : comment.text.replace(/\n/g, " ");

    const label = `${comment.file}:${pos}`;

    super(label, vscode.TreeItemCollapsibleState.None);

    this.description = preview;
    this.tooltip = new vscode.MarkdownString(
      `**#${comment.id}** — ${comment.file}:${pos}\n\n${comment.text}`
    );

    this.contextValue = comment.archived ? "archived" : "active";

    if (comment.archived) {
      this.iconPath = new vscode.ThemeIcon(
        "archive",
        new vscode.ThemeColor("disabledForeground")
      );
    } else {
      this.iconPath = new vscode.ThemeIcon("comment");
    }

    this.command = {
      command: "komments.goToComment",
      title: "Go to Comment",
      arguments: [this],
    };
  }
}

export class KommentsTreeProvider
  implements vscode.TreeDataProvider<CommentItem>
{
  private _onDidChangeTreeData = new vscode.EventEmitter<
    CommentItem | undefined | void
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  private showArchived = false;

  refresh(): void {
    this._onDidChangeTreeData.fire();
  }

  toggleArchived(): void {
    this.showArchived = !this.showArchived;
    this.refresh();
  }

  getTreeItem(element: CommentItem): vscode.TreeItem {
    return element;
  }

  getChildren(): CommentItem[] {
    try {
      const comments = cli.listComments(this.showArchived);
      return comments.map((c) => new CommentItem(c));
    } catch (e: any) {
      vscode.window.showErrorMessage(
        `Komments: failed to list comments — ${e.message}`
      );
      return [];
    }
  }
}
