---
name: Postman Collection Manager
description: Use this skill when the user asks to "add to postman", "update postman collection", "create postman requests", "sync postman", "add endpoints to postman", or after a spec document is created with new API endpoints.
version: 1.0.0
---

# Postman Collection Manager

Manage Postman collection endpoints using MCP tools.

## When This Skill Activates

- User says "add to postman" or "update postman"
- After a spec document is created with new endpoints
- When user wants to manage API collection
- During Phase 7 of the /dev workflow

---

## Prerequisites

This skill requires the **Postman MCP server** to be configured. The MCP tools (`mcp__postman__*`) must be available in the environment.

**To verify MCP availability:**
- Check if `mcp__postman__getCollection` tool is accessible
- If MCP tools are unavailable, the user should configure the Postman MCP server in their Claude Code settings

**Fallback behavior:**
- If MCP tools are not available, inform the user that manual Postman collection updates are required
- Provide the endpoint details in a format that can be manually imported into Postman

---

## Configuration

Use the workspace and collection IDs from CLAUDE.md:

| Resource   | Name      | ID                                             |
| ---------- | --------- | ---------------------------------------------- |
| Workspace  | golang    | `1ee078be-5479-45b9-9d5a-883cd4c6ef50`         |
| Collection | go-sample | `25403495-fd7a5765-5ad9-48d3-ab01-f3317012f96e` |

---

## Operations

### 1. Add Endpoints from Spec

**Trigger**: User wants to add endpoints from a spec file

**Steps**:
1. List available spec files in `spec/` directory
2. Ask user to select a spec file (or provide path)
3. Read the spec file and extract API endpoints from the "API Endpoints" section
4. For each endpoint, use `mcp__postman__createCollectionRequest` to add:
   - Name: Endpoint name from spec
   - Method: HTTP method (GET/POST/PUT/DELETE)
   - URL: `{{base_url}}/api/v1/...`
   - Headers: `Content-Type: application/json` for POST/PUT
   - Body: Sample request body from spec (if applicable)
5. Report success/failure for each endpoint

### 2. Add Single Endpoint

**Trigger**: User wants to add one endpoint manually

**Steps**:
1. Ask for endpoint details:
   - Name (e.g., "Create User")
   - Method (GET/POST/PUT/DELETE)
   - Path (e.g., `/api/v1/users`)
   - Auth required? (Yes/No)
   - Request body? (for POST/PUT)
2. Use `mcp__postman__createCollectionRequest` to add the endpoint
3. Confirm success

### 3. Add Endpoint to Folder

**Trigger**: User wants to add endpoint to a specific folder

**Steps**:
1. Use `mcp__postman__getCollection` to list existing folders
2. Ask user to select a folder (or create new one)
3. If new folder, use `mcp__postman__createCollectionFolder`
4. Follow "Add Single Endpoint" steps with `folderId` parameter

### 4. List Collection Structure

**Trigger**: User wants to see current collection structure

**Steps**:
1. Use `mcp__postman__getCollection` with `collectionId`
2. Present the structure in a tree format:
   ```
   go-sample/
   ├── Auth/
   │   ├── POST /api/v1/auth/register
   │   └── POST /api/v1/auth/login
   ├── Users/
   │   ├── GET /api/v1/users/me
   │   └── PUT /api/v1/users/me
   └── Voice Memos/
       ├── POST /api/v1/voice-memos
       └── GET /api/v1/voice-memos
   ```

### 5. Update Endpoint

**Trigger**: User wants to update an existing endpoint

**Steps**:
1. Use `mcp__postman__getCollection` to list endpoints
2. Ask user to select endpoint to update
3. Ask what to update (name, method, URL, body)
4. Use `mcp__postman__updateCollectionRequest` to update

### 6. Delete Endpoint

**Trigger**: User wants to remove an endpoint

**Steps**:
1. Use `mcp__postman__getCollection` to list endpoints
2. Ask user to select endpoint to delete
3. Confirm deletion
4. Use `mcp__postman__deleteCollectionRequest` to delete

---

## Request Templates

### Standard Request Structure

```json
{
  "name": "Endpoint Name",
  "request": {
    "method": "POST",
    "header": [
      {
        "key": "Content-Type",
        "value": "application/json"
      }
    ],
    "body": {
      "mode": "raw",
      "raw": "{\n  \"field\": \"value\"\n}",
      "options": {
        "raw": {
          "language": "json"
        }
      }
    },
    "url": {
      "raw": "{{base_url}}/api/v1/path",
      "host": ["{{base_url}}"],
      "path": ["api", "v1", "path"]
    }
  }
}
```

### Protected Endpoint (with Auth)

Add to header:
```json
{
  "key": "Authorization",
  "value": "Bearer {{access_token}}"
}
```

---

## Folder Organization

Follow the existing route group structure from the API:

| Folder         | Routes                  |
| -------------- | ----------------------- |
| Auth           | `/api/v1/auth/*`        |
| Users          | `/api/v1/users/*`       |
| Voice Memos    | `/api/v1/voice-memos/*` |
| Teams          | `/api/v1/teams/*`       |
| Invitations    | `/api/v1/invitations/*` |

When adding endpoints, place them in the appropriate folder based on their path.

---

## Error Handling

- If MCP tool fails, report the error and suggest alternatives
- If collection/folder not found, offer to create it
- If endpoint already exists (by name), ask if user wants to update or skip

---

## Summary

After any operation, provide a summary:

```markdown
## Postman Update Summary

**Operation**: [Add from Spec / Add Single / Update / Delete]

### Changes Made
- [Added/Updated/Deleted]: [Endpoint Name] - [METHOD] [URL]

### Collection Link
https://www.postman.com/[workspace]/[collection]
```
