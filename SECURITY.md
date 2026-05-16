# Security Policy

## Supported Versions

Until seed reaches 1.0, only the **latest released version** receives
security fixes. Older 0.x versions will not be patched — upgrade to the
current minor.

| Version       | Supported          |
| ------------- | ------------------ |
| Latest 0.x    | :white_check_mark: |
| Older 0.x     | :x:                |
| Future 1.x.x  | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**Preferred:** Use GitHub's private vulnerability reporting on this repo —
[Security tab → Report a vulnerability](https://github.com/krisarmstrong/seed/security/advisories/new).
That gives us a private channel scoped to maintainers, with built-in CVE
coordination and an audit trail.

**Alternative:** Email kris.armstrong@icloud.com.

Either way, please:

1. **Do NOT open a public issue** for security vulnerabilities
2. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment** within 48 hours
- **Initial assessment** within 7 days
- **Resolution timeline** communicated based on severity
- **Credit** in release notes (if desired)

### Severity Levels

| Level    | Description                         | Target Resolution |
| -------- | ----------------------------------- | ----------------- |
| Critical | Remote code execution, auth bypass  | 24-48 hours       |
| High     | Data exposure, privilege escalation | 7 days            |
| Medium   | Limited impact vulnerabilities      | 30 days           |
| Low      | Minor issues, hardening             | Next release      |

## Security Best Practices

When deploying The Seed:

### Network Security

- Deploy on isolated/management networks when possible
- Use firewall rules to restrict access to the web interface
- Consider VPN access for remote diagnostics

### Authentication

- Change default credentials immediately
- Use strong passwords (12+ characters)
- Rotate credentials periodically

### HTTPS

- Use valid TLS certificates in production
- Consider Let's Encrypt for public-facing deployments
- Self-signed certificates are acceptable for isolated networks

### Updates

- Keep The Seed updated to the latest version
- Subscribe to release notifications
- Review changelogs for security fixes

## Security Features

The Seed includes:

- HTTPS by default
- Password authentication
- JWT session management
- Rate limiting on auth endpoints
- No default open ports (except web interface)
- Minimal attack surface (single binary)

## Scope

The following are in scope for security reports:

- Authentication/authorization bypass
- Remote code execution
- Cross-site scripting (XSS)
- SQL/command injection
- Sensitive data exposure
- Privilege escalation

The following are out of scope:

- Denial of service (DoS)
- Social engineering
- Physical access attacks
- Issues requiring unlikely user interaction

## Acknowledgments

We appreciate security researchers who help keep The Seed secure. Contributors will be acknowledged in release notes
unless they prefer to remain anonymous.
