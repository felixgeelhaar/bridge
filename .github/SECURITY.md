# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

1. **Do NOT** create a public GitHub issue for security vulnerabilities
2. Email security concerns to the maintainers privately
3. Include as much detail as possible:
   - Type of vulnerability
   - Full paths of affected source files
   - Step-by-step instructions to reproduce
   - Proof of concept or exploit code (if available)
   - Potential impact

### What to Expect

- **Acknowledgment**: Within 48 hours of your report
- **Initial Assessment**: Within 1 week
- **Resolution Timeline**: Depends on severity
  - Critical: 24-72 hours
  - High: 1-2 weeks
  - Medium: 2-4 weeks
  - Low: Next release cycle

### Security Measures

This project implements:

- **Static Analysis**: gosec for Go security linting
- **Dependency Scanning**: govulncheck for known vulnerabilities
- **Secret Detection**: gitleaks to prevent credential leaks
- **Code Review**: All changes require review before merge
- **Signed Commits**: GPG-signed commits recommended

### Security Best Practices for Contributors

1. Never commit secrets, API keys, or credentials
2. Use environment variables for sensitive configuration
3. Follow the principle of least privilege
4. Validate all user inputs
5. Use parameterized queries for database operations
6. Keep dependencies updated
7. Sign your commits with GPG

## Security Tooling

### Local Security Scans

```bash
# Run all security scans
make security

# Individual scans
gosec ./...
govulncheck ./...
gitleaks detect --source=.
```

### CI/CD Integration

Security scans run automatically on:
- Every push to main/develop branches
- Every pull request
- All release builds

Results are uploaded as SARIF reports to GitHub Security tab.
