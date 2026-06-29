if vim.g.loaded_komments then
  return
end
vim.g.loaded_komments = true

vim.api.nvim_create_user_command("Komments", function()
  require("komments").open_list()
end, { desc = "Open Komments list" })
