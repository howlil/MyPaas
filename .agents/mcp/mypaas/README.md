# MyPaas MCP Server

This MCP server allows Antigravity/Cursor IDE to interact with your MyPaas installation directly via the MyPaas API. The AI can list, deploy, start, and stop your projects automatically.

## Setup Instructions

1. **Backend VM Configuration:**
   - Log into your MyPaas VM.
   - Edit the `.env` file and add a secure secret string:
     ```
     MYPAAS_API_TOKEN=super_secret_token_123
     ```
   - Restart the backend to apply changes:
     ```
     cd mypaas/backend && docker compose -f ../docker-compose.prod.yml restart mypaas-api
     ```

2. **Local Machine (Laptop) Configuration:**
   - Edit `.agents/mcp/mypaas/mcp_config.json` locally on your laptop.
   - Replace `<GANTI_DENGAN_TOKEN_ANDA>` with the exact token you set on the VM.
   - If your API URL is different, update `MYPAAS_URL`.

3. **Restart Antigravity / Cursor:**
   - The IDE will automatically read `mcp_config.json`, compile the Go MCP server dynamically using `go run`, and launch it.
   - Try asking your AI: "List all my projects on MyPaas" or "Deploy mypaas project".
