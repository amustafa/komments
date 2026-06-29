BINARY    := komments
BUILD_DIR := build
LINK_DIR  := $(HOME)/.local/bin
MCP_DIR   := mcp-server
MCP_ENTRY := $(MCP_DIR)/dist/index.js

CLAUDE_SETTINGS := $(HOME)/.claude/settings.json

.PHONY: build mcp link install-claude clean

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/komments/

mcp:
	cd $(MCP_DIR) && npm install && npm run build

link: build
	@mkdir -p $(LINK_DIR)
	ln -sf $(CURDIR)/$(BUILD_DIR)/$(BINARY) $(LINK_DIR)/$(BINARY)

install-claude: build mcp link
	@mkdir -p $(dir $(CLAUDE_SETTINGS))
	@node -e 'const fs=require("fs");const p=process.argv[1],e=process.argv[2],b=process.argv[3];let s={};try{s=JSON.parse(fs.readFileSync(p,"utf-8"))}catch{};if(!s.mcpServers)s.mcpServers={};s.mcpServers.komments={command:"node",args:[e],env:{KOMMENTS_BIN:b}};fs.writeFileSync(p,JSON.stringify(s,null,2)+"\n");console.log("Updated "+p)' "$(CLAUDE_SETTINGS)" "$(CURDIR)/$(MCP_ENTRY)" "$(CURDIR)/$(BUILD_DIR)/$(BINARY)"

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(MCP_DIR)/dist
