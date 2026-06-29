local M = {}

local function komments_bin()
  local cfg = require("komments.config")
  return cfg.options.bin or "komments"
end

local function run(args)
  local bin = komments_bin()
  local cmd = bin .. " " .. args
  local output = vim.fn.system(cmd)
  if vim.v.shell_error ~= 0 then
    return nil, vim.fn.trim(output)
  end
  return vim.fn.trim(output), nil
end

local function run_json(args)
  local output, err = run(args .. " --json")
  if err then
    return nil, err
  end
  if output == "" or output == "[]" then
    return {}, nil
  end
  local ok, decoded = pcall(vim.json.decode, output)
  if not ok then
    return nil, "failed to parse JSON: " .. tostring(decoded)
  end
  return decoded, nil
end

function M.get_project_root()
  local git_root = vim.fn.systemlist("git rev-parse --show-toplevel")[1]
  if vim.v.shell_error == 0 and git_root and git_root ~= "" then
    return git_root
  end
  return vim.fn.getcwd()
end

function M.relative_path(absolute_path, root)
  root = root or M.get_project_root()
  absolute_path = vim.fn.fnamemodify(absolute_path, ":p")
  if not root:match("/$") then
    root = root .. "/"
  end
  if absolute_path:sub(1, #root) == root then
    return absolute_path:sub(#root + 1)
  end
  return absolute_path
end

function M.add_comment(file, position, text)
  local pos_spec
  if position.type == "cursor" then
    pos_spec = tostring(position.line)
  else
    pos_spec = position.start_line .. "-" .. position.end_line
  end

  local escaped_text = vim.fn.shellescape(text)
  local escaped_file = vim.fn.shellescape(file)
  local output, err = run("add " .. escaped_file .. " " .. pos_spec .. " " .. escaped_text)
  if err then
    vim.notify("[komments] " .. err, vim.log.levels.ERROR)
    return nil
  end

  return { file = file, position = position, text = text }
end

function M.archive_comment(id)
  local _, err = run("archive " .. id)
  if err then
    vim.notify("[komments] " .. err, vim.log.levels.ERROR)
    return false
  end
  return true
end

function M.unarchive_comment(id)
  local _, err = run("unarchive " .. id)
  if err then
    vim.notify("[komments] " .. err, vim.log.levels.ERROR)
    return false
  end
  return true
end

function M.update_comment(id, new_text)
  local escaped = vim.fn.shellescape(new_text)
  local _, err = run("edit " .. id .. " " .. escaped)
  if err then
    vim.notify("[komments] " .. err, vim.log.levels.ERROR)
    return false
  end
  return true
end

function M.get_active_comments()
  local comments, err = run_json("list")
  if err then
    vim.notify("[komments] " .. err, vim.log.levels.ERROR)
    return {}
  end
  return comments or {}
end

function M.get_all_comments()
  local comments, err = run_json("list --all")
  if err then
    vim.notify("[komments] " .. err, vim.log.levels.ERROR)
    return {}
  end
  return comments or {}
end

return M
