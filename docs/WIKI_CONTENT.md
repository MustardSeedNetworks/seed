# GitHub Wiki Content Guide for The Seed

**Purpose:** This document describes the wiki structure for the GitHub Wiki at
<https://github.com/krisarmstrong/seed/wiki>

**Status:** Wiki content is maintained in `docs/wiki/` directory

---

## Wiki Pages

The wiki pages are located in the `docs/wiki/` directory:

| Page                 | File                                                | Description                    |
| -------------------- | --------------------------------------------------- | ------------------------------ |
| Home                 | [Home.md](wiki/Home.md)                             | Welcome page and quick links   |
| Installation (macOS) | [Installation-macOS.md](wiki/Installation-macOS.md) | macOS installation guide       |
| Installation (Linux) | [Installation-Linux.md](wiki/Installation-Linux.md) | Linux installation guide       |
| Quick Start          | [Quick-Start-Guide.md](wiki/Quick-Start-Guide.md)   | 5-minute getting started guide |
| Network Discovery    | [Network-Discovery.md](wiki/Network-Discovery.md)   | Device discovery documentation |
| FAQ                  | [FAQ.md](wiki/FAQ.md)                               | Frequently asked questions     |
| Sidebar              | [\_Sidebar.md](wiki/_Sidebar.md)                    | Wiki navigation sidebar        |

## Wiki Structure

```text
Home
├── Getting Started
│   ├── Installation (macOS)
│   ├── Installation (Linux)
│   ├── Installation (Docker)
│   ├── First-Time Setup
│   └── Quick Start Guide
├── Features
│   ├── Network Discovery
│   ├── WiFi Survey & Planning
│   ├── Speed Testing
│   ├── Cable Diagnostics
│   ├── DHCP Rogue Detection
│   ├── Vulnerability Scanning
│   └── Compliance Reporting
├── Configuration
│   ├── Network Interfaces
│   ├── SNMP Settings
│   ├── User Management
│   └── API Configuration
├── Troubleshooting
│   ├── Common Issues
│   ├── Error Messages
│   └── Performance Tuning
├── API Reference
│   ├── Authentication
│   ├── REST Endpoints
│   └── WebSocket Events
├── Development
│   ├── Building from Source
│   ├── Contributing
│   └── Development Environment
└── FAQ
```

---

## How to Populate the Wiki

### Step 1: Enable Wiki (Already Done)

The wiki is enabled. Now you need to add content.

### Step 2: Create Pages via GitHub Web Interface

1. Go to: <https://github.com/krisarmstrong/seed/wiki>

2. Click "Create the first page" (or "New Page" if one exists)

3. Copy/paste content from files in `docs/wiki/` directory

4. Create pages in this order:
   1. Home (required - first page)
   2. Installation-macOS
   3. Installation-Linux
   4. Quick-Start-Guide
   5. Network-Discovery
   6. FAQ
   7. (Continue with remaining pages as time permits)

### Step 3: Link Pages

Wiki pages auto-link via `[[Page-Name]]` syntax.

**Example:**

```markdown
See the [Installation Guide](Installation-macOS) for setup instructions.
```

or

```markdown
See the [[Installation-macOS|Installation Guide]] for setup instructions.
```

### Step 4: Add Sidebar

Create a page named `_Sidebar.md` using content from `docs/wiki/_Sidebar.md`.

---

## Next Steps

1. **Create wiki pages** using content from `docs/wiki/` directory
2. **Add screenshots** (wiki supports image uploads)
3. **Link to wiki** from README.md
4. **Update as product evolves**

---

**Document Owner:** Kris Armstrong **Last Updated:** December 2025
