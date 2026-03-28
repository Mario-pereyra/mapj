# Confluence Auth — Detailed Reference

---

## Auth Type Decision Tree

```
Your Confluence URL?
│
├─ Contains "atlassian.net" (e.g., company.atlassian.net)
│   └─ Use BASIC AUTH
│       Required: email address + Atlassian API token
│       mapj auth login confluence \
│         --url https://company.atlassian.net \
│         --username your-email@company.com \
│         --token YOUR_API_TOKEN
│
├─ Contains "tdninterno.totvs.com" or similar internal Server/DC
│   └─ Use BEARER PAT
│       Required: Personal Access Token only
│       mapj auth login confluence \
│         --url https://tdninterno.totvs.com \
│         --token YOUR_PAT
│       ⚠️ NEVER add --username → activates Basic Auth → 401!
│
└─ Public (tdn.totvs.com)
    └─ No auth needed for public content
        mapj confluence export "https://tdn.totvs.com/display/..."
        (CLI falls back to HTML scraping if API is unavailable)
```

---

## Auto-detection Logic

The CLI auto-detects auth type on login:

```
URL contains "atlassian.net"?
├─ YES → authType = "basic" stored in credentials.enc
└─ NO  → authType = "bearer" stored in credentials.enc
```

Override with `--auth-type basic|bearer` if auto-detection is wrong.

---

## The 401 Bug — Root Cause and Fix

**The bug (old behavior):** If `--username` was passed, the CLI assumed Basic Auth regardless of URL. This caused 401 on Server/DC instances that require Bearer.

**The fix (current behavior):** `authType` is stored in credentials. `getConfluenceClient()` reads the stored `authType`, not the presence of a username.

**If you get 401:**
```bash
# Old credentials may have wrong authType stored
# Re-login without --username to fix
mapj auth login confluence \
  --url https://tdninterno.totvs.com \
  --token YOUR_PAT_TOKEN
```

---

## Confluence Cloud — Getting an API Token

1. Go to: https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Copy the token (shown only once)
4. Use as `--token` with `--username YOUR_EMAIL`

---

## Confluence Server/DC — Getting a PAT

1. Go to: `https://your-instance.com/profile.action` (your user profile)
2. Click "Personal Access Tokens" in the left sidebar
3. Create a new token
4. Copy the token (shown only once)
5. Use as `--token` only (no `--username`)

---

## Verifying Auth is Working

```bash
# Check stored credentials
mapj auth status
# → "Confluence: ✓ authenticated"

# Test with a simple export (does not save file)
mapj confluence export 22479548
# If response contains "ok": true → auth works
# If 401 → re-login with correct auth type
```
