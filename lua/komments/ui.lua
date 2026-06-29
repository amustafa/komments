local store = require("komments.store")
local config = require("komments.config")

local M = {}

--- Open a floating input window for adding a new comment.
---@param file string relative file path
---@param position table position data
function M.open_input(file, position)
  local opts = config.options.ui.input
  local buf = vim.api.nvim_create_buf(false, true)
  vim.bo[buf].buftype = "nofile"
  vim.bo[buf].filetype = "markdown"

  local win = vim.api.nvim_open_win(buf, true, {
    relative = "cursor",
    row = 1,
    col = 0,
    width = opts.width,
    height = opts.height,
    style = "minimal",
    border = opts.border,
    title = " Komment ",
    title_pos = "center",
  })

  vim.cmd("startinsert")

  local function close_float()
    if vim.api.nvim_win_is_valid(win) then
      vim.api.nvim_win_close(win, true)
    end
    if vim.api.nvim_buf_is_valid(buf) then
      vim.api.nvim_buf_delete(buf, { force = true })
    end
  end

  -- Submit: Ctrl-Enter in insert mode, or Enter in normal mode
  local function submit()
    local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
    local text = vim.fn.trim(table.concat(lines, "\n"))
    if text ~= "" then
      store.add_comment(file, position, text)
      vim.notify("[komments] Comment added", vim.log.levels.INFO)
    end
    close_float()
  end

  local function cancel()
    close_float()
  end

  vim.keymap.set("i", "<C-CR>", submit, { buffer = buf })
  vim.keymap.set("n", "<CR>", submit, { buffer = buf })
  vim.keymap.set("n", "q", cancel, { buffer = buf })
  vim.keymap.set("n", "<Esc>", cancel, { buffer = buf })

  -- Close on BufLeave
  vim.api.nvim_create_autocmd("BufLeave", {
    buffer = buf,
    once = true,
    callback = close_float,
  })
end

--- Format a position for display.
---@param pos table
---@return string
local function format_position(pos)
  if pos.type == "cursor" then
    return tostring(pos.line)
  else
    return pos.start_line .. "-" .. pos.end_line
  end
end

--- Open the comments list view.
function M.open_list()
  local opts = config.options.ui.list
  local comments = store.get_all_comments()

  local editor_w = vim.o.columns
  local editor_h = vim.o.lines
  local width = math.floor(editor_w * opts.width)
  local height = math.floor(editor_h * opts.height)
  local row = math.floor((editor_h - height) / 2)
  local col = math.floor((editor_w - width) / 2)

  local buf = vim.api.nvim_create_buf(false, true)
  vim.bo[buf].buftype = "nofile"
  vim.bo[buf].filetype = "komments"

  -- Build display lines and track comment IDs by line number
  local lines = {}
  local id_by_line = {}
  local hl_ranges = {} -- { line, archived }

  if #comments == 0 then
    table.insert(lines, "  No comments yet. Press q to close.")
  else
    for _, c in ipairs(comments) do
      local prefix = c.archived and "[archived] " or ""
      local pos_str = format_position(c.position)
      local text_preview = c.text:gsub("\n", " ")
      if #text_preview > (width - 30) then
        text_preview = text_preview:sub(1, width - 33) .. "..."
      end
      local line = string.format("  [#%d] %s:%s — %s%s", c.id, c.file, pos_str, prefix, text_preview)
      local line_idx = #lines
      table.insert(lines, line)
      id_by_line[line_idx] = c.id
      table.insert(hl_ranges, { line = line_idx, archived = c.archived })
    end
  end

  vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
  vim.bo[buf].modifiable = false

  local win = vim.api.nvim_open_win(buf, true, {
    relative = "editor",
    row = row,
    col = col,
    width = width,
    height = height,
    style = "minimal",
    border = opts.border,
    title = " Komments ",
    title_pos = "center",
  })

  -- Highlight archived lines
  local ns = vim.api.nvim_create_namespace("komments_list")
  for _, hl in ipairs(hl_ranges) do
    if hl.archived then
      vim.api.nvim_buf_add_highlight(buf, ns, "Comment", hl.line, 0, -1)
    end
  end

  -- Close
  local function close()
    vim.api.nvim_win_close(win, true)
    vim.api.nvim_buf_delete(buf, { force = true })
  end

  -- Get comment ID at current cursor line
  local function current_id()
    local line = vim.api.nvim_win_get_cursor(win)[1] - 1 -- 0-indexed
    return id_by_line[line]
  end

  -- Refresh the list
  local function refresh()
    close()
    M.open_list()
  end

  -- Archive current comment
  local function archive()
    local id = current_id()
    if id then
      store.archive_comment(id)
      vim.notify("[komments] Comment #" .. id .. " archived", vim.log.levels.INFO)
      refresh()
    end
  end

  -- Unarchive current comment
  local function unarchive()
    local id = current_id()
    if id then
      store.unarchive_comment(id)
      vim.notify("[komments] Comment #" .. id .. " unarchived", vim.log.levels.INFO)
      refresh()
    end
  end

  -- Edit current comment
  local function edit()
    local id = current_id()
    if not id then return end
    local all = store.get_all_comments()
    for _, c in ipairs(all) do
      if c.id == id then
        close()
        M.open_edit(c)
        return
      end
    end
  end

  -- Jump to comment location
  local function jump()
    local id = current_id()
    if not id then return end
    local all = store.get_all_comments()
    for _, c in ipairs(all) do
      if c.id == id then
        close()
        local root = store.get_project_root()
        local filepath = root .. "/" .. c.file
        vim.cmd("edit " .. vim.fn.fnameescape(filepath))
        local line = c.position.type == "cursor" and c.position.line or c.position.start_line
        vim.api.nvim_win_set_cursor(0, { line, 0 })
        return
      end
    end
  end

  vim.keymap.set("n", "q", close, { buffer = buf })
  vim.keymap.set("n", "<Esc>", close, { buffer = buf })
  vim.keymap.set("n", "a", archive, { buffer = buf })
  vim.keymap.set("n", "dd", archive, { buffer = buf })
  vim.keymap.set("n", "u", unarchive, { buffer = buf })
  vim.keymap.set("n", "<CR>", edit, { buffer = buf })
  vim.keymap.set("n", "e", edit, { buffer = buf })
  vim.keymap.set("n", "gd", jump, { buffer = buf })
end

--- Open a floating edit window pre-filled with existing comment text.
---@param comment table the comment to edit
function M.open_edit(comment)
  local opts = config.options.ui.input
  local buf = vim.api.nvim_create_buf(false, true)
  vim.bo[buf].buftype = "nofile"
  vim.bo[buf].filetype = "markdown"

  -- Pre-fill with existing text
  local existing_lines = vim.split(comment.text, "\n")
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, existing_lines)

  local win = vim.api.nvim_open_win(buf, true, {
    relative = "editor",
    row = math.floor((vim.o.lines - opts.height) / 2),
    col = math.floor((vim.o.columns - opts.width) / 2),
    width = opts.width,
    height = opts.height,
    style = "minimal",
    border = opts.border,
    title = string.format(" Edit Komment #%d ", comment.id),
    title_pos = "center",
  })

  local function submit()
    local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
    local text = vim.fn.trim(table.concat(lines, "\n"))
    if text ~= "" then
      store.update_comment(comment.id, text)
      vim.notify("[komments] Comment #" .. comment.id .. " updated", vim.log.levels.INFO)
    end
    vim.api.nvim_win_close(win, true)
    vim.api.nvim_buf_delete(buf, { force = true })
  end

  local function cancel()
    vim.api.nvim_win_close(win, true)
    vim.api.nvim_buf_delete(buf, { force = true })
  end

  vim.keymap.set("i", "<C-CR>", submit, { buffer = buf })
  vim.keymap.set("n", "<CR>", submit, { buffer = buf })
  vim.keymap.set("n", "q", cancel, { buffer = buf })
  vim.keymap.set("n", "<Esc>", cancel, { buffer = buf })
end

return M
