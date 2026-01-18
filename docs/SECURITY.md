# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them by opening a security advisory on GitHub:
https://github.com/chris-roerig/homegit/security/advisories/new

Or contact via GitHub: [@chris-roerig](https://github.com/chris-roerig)

You should receive a response within 48 hours.

Please include the following information:

- Type of issue (e.g., buffer overflow, authentication bypass, etc.)
- Full paths of source file(s) related to the issue
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

## Security Considerations

⚠️ **Important**: homegit has **NO authentication** by design. It is intended for:

- Personal use on trusted networks
- Offline/local development
- Home networks where you control all devices

**Do NOT:**
- Expose homegit to the public internet
- Use it on untrusted networks
- Store sensitive/production code without additional security measures

**Recommendations:**
- Run on localhost or private networks only
- Use firewall rules to restrict access
- Consider SSH tunneling for remote access
- Use VPN for access from outside your network

## Disclosure Policy

When we receive a security bug report, we will:

1. Confirm the problem and determine affected versions
2. Audit code to find similar problems
3. Prepare fixes for all supported versions
4. Release new versions as soon as possible

## Comments on this Policy

If you have suggestions on how this process could be improved, please submit a pull request.
