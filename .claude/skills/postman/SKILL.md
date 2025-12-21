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
4. Use `mcp__postman__patchCollection` to add fully-configured endpoints:
   - Include `info` object with collection name and schema
   - Include `item` array with each endpoint's full configuration:
     - Name: Endpoint name from spec
     - Method: HTTP method (GET/POST/PUT/DELETE)
     - URL: `{{base_url}}/api/v1/...` with path segments
     - Headers: `Content-Type: application/json` for POST/PUT, `Authorization` for protected routes
     - Body: Sample request body from spec (if applicable)
5. Report success/failure for each endpoint

**Note:** Do NOT use `createCollectionRequest` - it cannot set URL, headers, or body. See "MCP Tool Limitations" section.

### 2. Add Single Endpoint

**Trigger**: User wants to add one endpoint manually

**Steps**:
1. Ask for endpoint details:
   - Name (e.g., "Create User")
   - Method (GET/POST/PUT/DELETE)
   - Path (e.g., `/api/v1/users`)
   - Auth required? (Yes/No)
   - Request body? (for POST/PUT)
2. Use `mcp__postman__patchCollection` with full endpoint configuration
3. Confirm success

**Note:** Do NOT use `createCollectionRequest` for configured endpoints. See "MCP Tool Limitations" section.

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

## MCP Tool Limitations & Workarounds

### Tool Limitations

| Tool | Supported Parameters | NOT Supported |
|------|---------------------|---------------|
| `createCollectionRequest` | `name`, `collectionId`, `folderId` | URL, headers, body, method |
| `updateCollectionRequest` | `name`, `method`, `requestId`, `collectionId` | URL, headers, body |
| `patchCollection` | Collection-level: name, events, variables | Request items (use `putCollection`) |
| `putCollection` | Full collection schema with all items | - |

### Workaround: Use `putCollection` for Full Request Configuration

Since `createCollectionRequest` and `updateCollectionRequest` have limited parameters, use `putCollection` to add/update fully-configured endpoints with URL, headers, and body.

**Important:** `patchCollection` only updates collection-level properties (name, events, variables) - it does NOT add or update request items. Use `putCollection` instead.

**Required Structure:**

When using `putCollection`, you MUST:
1. Include the `info` object with `name` and `schema`
2. Include ALL existing items (with their IDs) to preserve them
3. Add/update items with full request configuration

```json
{
  "collectionId": "25403495-fd7a5765-5ad9-48d3-ab01-f3317012f96e",
  "collection": {
    "info": {
      "name": "go-sample",
      "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
    },
    "item": [
      {
        "name": "Endpoint Name",
        "request": {
          "method": "POST",
          "header": [
            {"key": "Content-Type", "value": "application/json"},
            {"key": "Authorization", "value": "Bearer {{access_token}}"}
          ],
          "url": {
            "raw": "{{base_url}}/api/v1/path/:id",
            "host": ["{{base_url}}"],
            "path": ["api", "v1", "path", ":id"]
          }
        }
      }
    ]
  }
}
```

**Common Error:**

If you omit the `info` object, you'll get:
```
API request failed: 400 {"error":{"name":"paramMissingError","message":"'info' key is required when 'collection' is empty."}}
```

### Recommended Approach for Adding Endpoints

1. **For basic endpoint creation (name only):** Use `createCollectionRequest`
2. **For fully-configured endpoints (URL, headers, body):** Use `putCollection`
3. **For updating name/method only:** Use `updateCollectionRequest`
4. **For updating URL/headers/body:** Use `putCollection`

**Workflow for adding new endpoints:**
1. First, get the full collection with `getCollection` (model=full)
2. Add new items to the item array with full configuration
3. Use `putCollection` with the complete collection (preserving existing item IDs)

### Example: Adding New Endpoints with `putCollection`

```javascript
// Use putCollection - must include ALL existing items to preserve them
{
  "collectionId": "25403495-fd7a5765-5ad9-48d3-ab01-f3317012f96e",
  "collection": {
    "info": {
      "name": "go-sample",
      "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
    },
    "variable": [
      {"key": "base_url", "value": "http://localhost:8080"},
      {"key": "access_token", "value": ""}
    ],
    "item": [
      // ... existing items with their IDs preserved ...
      {"id": "existing-item-id", "name": "Existing Endpoint", ...},

      // New items (no ID = will be created)
      {
        "name": "[Private Memos] Confirm Upload",
        "request": {
          "method": "POST",
          "header": [
            {"key": "Authorization", "value": "Bearer {{access_token}}"}
          ],
          "url": {
            "raw": "{{base_url}}/api/v1/voice-memos/:id/confirm-upload",
            "host": ["{{base_url}}"],
            "path": ["api", "v1", "voice-memos", ":id", "confirm-upload"],
            "variable": [{"key": "id"}]
          }
        }
      }
    ]
  }
}
```

**Warning:** If you omit existing items from the `item` array, they will be DELETED.

---

## Error Handling

- If MCP tool fails, report the error and suggest alternatives
- If collection/folder not found, offer to create it
- If endpoint already exists (by name), ask if user wants to update or skip
- **If `createCollectionRequest` fails to set URL/headers:** Use `patchCollection` instead (see limitations above)

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
