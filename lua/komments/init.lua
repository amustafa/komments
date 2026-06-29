local M = {}

--- Initialize the plugin with user options.
---@param opts? table
function M.setup(opts)
  local config = require("komments.config")
  config.setup(opts)

  local keymap = config.options.keymap
  local list_keymap = config.options.list_keymap

  -- Normal mode: comment at cursor
  vim.keymap.set("n", keymap, function()
    M.add_comment_at_cursor()
  end, { desc = "Komments: add comment at cursor" })

  -- Visual mode: comment at selection
  vim.keymap.set("v", keymap, function()
    -- Exit visual mode first so '< and '> marks are set
    vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes("<Esc>", true, false, true), "nx", false)
    vim.schedule(function()
      M.add_comment_at_selection()
    end)
  end, { desc = "Komments: add comment at selection" })

  -- List view
  vim.keymap.set("n", list_keymap, function()
    M.open_list()
  end, { desc = "Komments: open comment list" })
end

--- Add a comment at the current cursor position.
function M.add_comment_at_cursor()
  local store = require("komments.store")
  local ui = require("komments.ui")

  local file = store.relative_path(vim.api.nvim_buf_get_name(0))
  local cursor = vim.api.nvim_win_get_cursor(0)
  local position = {
    type = "cursor",
    line = cursor[1],
    col = cursor[2] + 1, -- convert 0-indexed col to 1-indexed
  }

  ui.open_input(file, position)
end

--- Add a comment at the visual selection.
function M.add_comment_at_selection()
  local store = require("komments.store")
  local ui = require("komments.ui")

  local file = store.relative_path(vim.api.nvim_buf_get_name(0))
  local start_pos = vim.api.nvim_buf_get_mark(0, "<")
  local end_pos = vim.api.nvim_buf_get_mark(0, ">")
  local position = {
    type = "range",
    start_line = start_pos[1],
    start_col = start_pos[2] + 1,
    end_line = end_pos[1],
    end_col = end_pos[2] + 1,
  }

  ui.open_input(file, position)
end

--- Open the comments list view.
function M.open_list()
  local ui = require("komments.ui")
  ui.open_list()
end

return M
