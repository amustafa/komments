import { execFileSync } from "child_process";
import * as vscode from "vscode";
import type { Comment } from "./types.js";

function getBin(): string {
  return vscode.workspace.getConfiguration("komments").get("bin", "komments");
}

function getCwd(): string {
  const folders = vscode.workspace.workspaceFolders;
  if (folders && folders.length > 0) {
    return folders[0].uri.fsPath;
  }
  return process.cwd();
}

function run(...args: string[]): string {
  return execFileSync(getBin(), args, {
    encoding: "utf-8",
    cwd: getCwd(),
    stdio: ["pipe", "pipe", "pipe"],
  }).trim();
}

export function listComments(includeArchived: boolean): Comment[] {
  const args = ["list", "--json"];
  if (includeArchived) args.push("--all");
  const output = run(...args);
  if (!output || output === "[]") return [];
  return JSON.parse(output) as Comment[];
}

export function getComment(id: number): Comment | undefined {
  try {
    const output = run("get", String(id), "--json");
    return JSON.parse(output) as Comment;
  } catch {
    return undefined;
  }
}

export function addComment(file: string, position: string, text: string): void {
  run("add", file, position, text);
}

export function editComment(id: number, text: string): void {
  run("edit", String(id), text);
}

export function archiveComment(id: number): void {
  run("archive", String(id));
}

export function unarchiveComment(id: number): void {
  run("unarchive", String(id));
}

export function deleteComment(id: number): void {
  run("delete", String(id));
}
