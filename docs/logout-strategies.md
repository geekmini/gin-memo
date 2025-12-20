# Logout Strategies

This document compares two approaches for implementing secure logout in JWT-based authentication systems.

## Overview

| Approach                 | Server-Side Instant | Client-Side Instant | Stolen Token Protected |
| ------------------------ | ------------------- | ------------------- | ---------------------- |
| Token Versioning + SSE   | Yes                 | Yes                 | Yes                    |
| Access/Refresh + Polling | No (15 min window)  | ~5 sec delay        | No                     |

## Approach 1: Token Versioning + SSE

### Architecture

```mermaid
graph TB
    subgraph "Logout System"
        subgraph "Layer 1: Token Versioning"
            TV1[Server-side instant rejection]
            TV2[Stored in Redis + MongoDB]
            TV3[Checked on EVERY API request]
        end

        subgraph "Layer 2: SSE Push"
            SSE1[Client-side instant notification]
            SSE2[Notifies online devices immediately]
            SSE3[Validates token on reconnect]
        end
    end

    TV1 --> TV2 --> TV3
    SSE1 --> SSE2 --> SSE3
```

### How It Works

**Token Versioning:**

- Each user has a `tokenVersion` field (starts at 1)
- JWT includes `tokenVersion` in claims at login time
- On logout, server increments `tokenVersion`
- Auth middleware checks JWT version against stored version on every request
- Mismatch = 401 Unauthorized (instant rejection)

**SSE (Server-Sent Events):**

- Client opens persistent connection: `GET /api/v1/auth/events`
- Server pushes "logout" event when logout occurs
- Client receives event and logs out locally (instant notification)
- On reconnect, server validates token before accepting connection

### Data Model

```go
// User model
type User struct {
    ID           string `bson:"_id"`
    Email        string `bson:"email"`
    Password     string `bson:"password"`
    TokenVersion int    `bson:"tokenVersion"` // incremented on logout
}

// JWT claims
type Claims struct {
    UserID       string `json:"userId"`
    TokenVersion int    `json:"tokenVersion"` // must match stored version
    jwt.RegisteredClaims
}
```

### Flow Diagrams

**Login Flow:**

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as MongoDB

    C->>S: POST /api/v1/auth/login
    S->>DB: Validate credentials
    DB-->>S: User (tokenVersion = 1)
    S->>S: Generate JWT with tokenVersion: 1
    S-->>C: Return JWT
```

**Logout Flow:**

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant R as Redis
    participant DB as MongoDB
    participant SSE as SSE Connections

    C->>S: POST /api/v1/auth/logout
    S->>DB: Increment tokenVersion (1 → 2)
    S->>R: Update tokenVersion cache
    S->>SSE: Push "logout" event to all connections
    S-->>C: 204 No Content
```

**API Request Flow (Auth Middleware):**

```mermaid
flowchart TD
    A[Protected API Request] --> B[Extract tokenVersion from JWT]
    B --> C{Check Redis cache}
    C -->|Hit| D[Get stored tokenVersion]
    C -->|Miss| E[Fetch from MongoDB]
    E --> F[Cache in Redis]
    F --> D
    D --> G{JWT version == Stored version?}
    G -->|Yes| H[Continue to handler]
    G -->|No| I[401 Unauthorized]
```

**SSE Connection Flow:**

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    C->>S: GET /api/v1/auth/events (with JWT)
    S->>S: Validate JWT + tokenVersion

    alt Invalid token
        S-->>C: 401 Unauthorized
    else Valid token
        S-->>C: Hold connection open
        Note over C,S: Connection stays open...
        S->>C: event: logout<br/>data: {}
        C->>C: Discard token, redirect to login
    end
```

**Offline Device Reconnect:**

```mermaid
sequenceDiagram
    participant D as Device (was offline)
    participant S as Server

    Note over D: Device comes online
    D->>S: Reconnect SSE (with old JWT)
    S->>S: Validate tokenVersion
    Note over S: JWT version (1) != Stored version (2)
    S-->>D: 401 Unauthorized
    D->>D: Force logout locally
```

### Scenario Coverage

| Scenario                           | What Catches It                    |
| ---------------------------------- | ---------------------------------- |
| Device online at logout time       | SSE push (instant)                 |
| Device offline, comes online       | SSE reconnect validation (instant) |
| Device makes API call after logout | Token versioning (instant)         |
| All else fails                     | Token expiry (eventual)            |

### Pros and Cons

**Pros:**

- True instant logout (server + client)
- Stolen tokens rejected immediately
- Single token type (simpler)
- "Logout everywhere" built-in

**Cons:**

- Redis/DB check on every request
- Requires SSE infrastructure
- Single logout scope (all devices at once)

---

## Approach 2: Access Token + Refresh Token + Polling

### Architecture

```mermaid
graph TB
    subgraph "Logout System"
        subgraph "Access Token - 15 min TTL"
            AT1[Stateless validation]
            AT2[No DB lookup on API requests]
            AT3[Short-lived, disposable]
        end

        subgraph "Refresh Token - 7 days TTL"
            RT1[Stateful - stored in Redis/MongoDB]
            RT2[Deleted on logout]
            RT3[Only sent to /auth/refresh]
        end

        subgraph "Client Polling - every 5 sec"
            CP1[Checks session validity]
            CP2[Notifies client of logout]
        end
    end

    AT1 --> AT2 --> AT3
    RT1 --> RT2 --> RT3
    CP1 --> CP2
```

### How It Works

**Two Token Types:**

- Access Token: Short-lived (15 min), stateless, used for API calls
- Refresh Token: Long-lived (7 days), stateful, used only to get new access tokens

**Logout:**

- Delete refresh token from server
- Access token remains valid until expiry (up to 15 min)
- Client polling detects logout and discards tokens locally

**Polling:**

- Client calls `/auth/session-valid` every N seconds
- Server checks if refresh token exists
- If not, client logs out locally

### Data Model

```go
// Access Token claims (stateless)
type AccessClaims struct {
    UserID string `json:"userId"`
    jwt.RegisteredClaims // includes exp
}

// Refresh Token (stored in Redis/MongoDB)
type RefreshToken struct {
    Token     string    `bson:"token"`
    UserID    string    `bson:"userId"`
    ExpiresAt time.Time `bson:"expiresAt"`
}
```

### Flow Diagrams

**Login Flow:**

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as MongoDB/Redis

    C->>S: POST /api/v1/auth/login
    S->>S: Validate credentials
    S->>S: Generate Access Token (15 min)
    S->>S: Generate Refresh Token (7 days)
    S->>DB: Store Refresh Token
    S-->>C: Return both tokens
```

**API Request Flow:**

```mermaid
flowchart TD
    A[Protected API Request] --> B[Extract Access Token]
    B --> C[Validate JWT signature + expiry]
    C --> D{Valid?}
    D -->|Yes| E[Continue to handler]
    D -->|No/Expired| F[401 Unauthorized]
    Note1[No database lookup - stateless!]

    style Note1 fill:#fff3cd
```

**Token Refresh Flow:**

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as Redis/MongoDB

    C->>S: POST /api/v1/auth/refresh
    Note over C,S: Body: { refreshToken: "rf_8a7b3c..." }
    S->>DB: Check refresh token exists?

    alt Valid refresh token
        DB-->>S: Token found
        S->>S: Generate new Access Token
        S-->>C: New Access Token
    else Invalid/Expired
        DB-->>S: Not found
        S-->>C: 401 Unauthorized
    end
```

**Logout Flow:**

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as Redis/MongoDB

    C->>S: POST /api/v1/auth/logout
    S->>DB: Delete refresh token
    S-->>C: 204 No Content

    Note over C,S: Access token still valid for up to 15 min!
```

**Polling Flow:**

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as Redis/MongoDB

    loop Every 5 seconds
        C->>S: GET /api/v1/auth/session-valid
        S->>DB: Check refresh token exists?

        alt Exists
            S-->>C: { valid: true }
        else Gone (logged out)
            S-->>C: { valid: false }
            C->>C: Discard tokens, redirect to login
        end
    end
```

### Timeline After Logout

```mermaid
gantt
    title Timeline After Logout
    dateFormat HH:mm:ss
    axisFormat %H:%M:%S

    section Logout Event
    User logs out, refresh token deleted :milestone, 10:00:00, 0s

    section Honest Client
    Client polls, discovers logout       :done, 10:00:00, 3s
    Client logged out locally            :milestone, 10:00:03, 0s

    section Attacker (stolen token)
    Access token still works             :crit, 10:00:00, 15m
    Access token expires                 :milestone, 10:15:00, 0s
    Attacker finally blocked             :done, 10:15:00, 1s
```

```mermaid
flowchart LR
    subgraph "10:00:00"
        A[Logout] --> B[Refresh token deleted]
    end

    subgraph "10:00:03"
        C[Client polls] --> D[Discovers logout]
        D --> E[Honest client logged out]
    end

    subgraph "10:00:03 - 10:15:00"
        F[Attacker with stolen token] --> G[Can still make API calls!]
    end

    subgraph "10:15:00"
        H[Access token expires] --> I[Attacker blocked]
    end

    B --> C
    B --> F
    G --> H
```

### Pros and Cons

**Pros:**

- Stateless API requests (better performance)
- Horizontal scaling easier (no shared state per request)
- Industry standard approach

**Cons:**

- 15-minute vulnerability window after logout
- Stolen tokens work until access token expires
- Polling adds network overhead
- Two token types (more complexity)
- Client notification delayed by poll interval

---

## Detailed Comparison

### Security

| Aspect                            | Token Versioning + SSE | Access/Refresh + Polling   |
| --------------------------------- | ---------------------- | -------------------------- |
| Stolen token blocked after logout | Instantly              | Up to 15 min               |
| Server trusts old tokens          | Never                  | Until access token expires |
| Attack window                     | 0 seconds              | Up to 15 minutes           |

### Performance

| Aspect                   | Token Versioning + SSE | Access/Refresh + Polling  |
| ------------------------ | ---------------------- | ------------------------- |
| DB check per API request | Yes (Redis ~1ms)       | No (stateless)            |
| Network overhead         | SSE connection         | Poll every N seconds      |
| Scalability              | Redis is bottleneck    | Better horizontal scaling |

### User Experience

| Aspect                    | Token Versioning + SSE | Access/Refresh + Polling       |
| ------------------------- | ---------------------- | ------------------------------ |
| Client notified of logout | Instantly (SSE push)   | ~5 sec delay (poll interval)   |
| Offline device handling   | Caught on reconnect    | Caught on poll after reconnect |
| Logout scope              | All devices at once    | All devices at once            |

### Implementation Complexity

| Aspect                  | Token Versioning + SSE | Access/Refresh + Polling |
| ----------------------- | ---------------------- | ------------------------ |
| Token types             | 1                      | 2                        |
| Infrastructure needed   | Redis + SSE            | Redis + Polling endpoint |
| Auth middleware changes | Check version          | Standard JWT validation  |
| Client changes          | SSE connection         | Polling loop             |

---

## Decision Guide

### Choose Token Versioning + SSE if:

- Security is top priority
- You need instant logout (server + client)
- You cannot tolerate stolen tokens working after logout
- You already have Redis infrastructure
- Use cases: Banking, healthcare, enterprise apps

### Choose Access/Refresh + Polling if:

- Performance is top priority
- You need stateless API validation
- 15-minute vulnerability window is acceptable
- You want industry-standard approach
- Use cases: Social apps, content platforms, general web apps

---

## Hybrid Approach (Advanced)

For maximum flexibility, combine both:

```mermaid
graph TB
    subgraph "Hybrid Logout System"
        AT[Access Token - 15 min<br/>Stateless for most endpoints]
        RT[Refresh Token - 7 days<br/>Stateful, revocable]
        TV[Token Versioning<br/>Only for sensitive endpoints]
        SSE[SSE<br/>Real-time logout notification]
    end

    AT --> RT
    RT --> TV
    TV --> SSE
```

**How it works:**

- Regular endpoints: Stateless access token validation (fast)
- Sensitive endpoints: Check token version (secure)
- Logout: Revoke refresh token + increment version + SSE push

**Sensitive endpoints examples:**

- Password change
- Payment processing
- Account deletion
- Admin actions

**Hybrid Flow:**

```mermaid
flowchart TD
    A[API Request] --> B{Sensitive endpoint?}
    B -->|No| C[Validate JWT signature only<br/>Stateless - fast]
    B -->|Yes| D[Validate JWT + tokenVersion<br/>Check Redis - secure]
    C --> E[Continue to handler]
    D --> F{Version match?}
    F -->|Yes| E
    F -->|No| G[401 Unauthorized]
```

This gives you performance for most requests and instant revocation for critical actions.
