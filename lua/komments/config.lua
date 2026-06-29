local M = {}

M.defaults = {
  bin = "komments",
  keymap = "<leader>kc",
  list_keymap = "<leader>kl",
  ui = {
    input = {
      width = 60,
      height = 5,
      border = "rounded",
    },
    list = {
      width = 0.8,  -- fraction of editor width
      height = 0.6, -- fraction of editor height
      border = "rounded",
    },
  },
}

M.options = vim.deepcopy(M.defaults)

--- Merge user options with defaults.
---@param opts? table
function M.setup(opts)
  M.options = vim.tbl_deep_extend("force", vim.deepcopy(M.defaults), opts or {})
end

return M
