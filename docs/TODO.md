# Technical Debt & Future Improvements

This document tracks planned improvements and technical debt that requires preparation before implementation.

## Security

### Enable Strict `gosec` Failing in CI

**Status**: TODO (requires noise reduction first)
**Priority**: High
**Tracking**: #553

**Current State**:
- `gosec` runs in CI with `-no-fail` flag (`.github/workflows/ci.yml` line 160)
- Security findings are reported but don't block builds
- Results are uploaded to GitHub Security tab via SARIF

**Goal**:
Enable `gosec` to fail CI builds on high-confidence security findings, aligning with enterprise security standards.

**Prerequisites** (must complete first):
1. **Audit Current Findings**: Run `gosec ./...` locally and document all findings
2. **Categorize Issues**: Separate true positives from false positives
3. **Address True Positives**: Fix legitimate security issues
4. **Configure Exclusions**: Add `#nosec` comments with justification for known false positives
5. **Reduce Noise**: Ensure clean baseline with minimal findings

**Implementation Steps** (after prerequisites):
1. Remove `-no-fail` from gosec command:
   ```yaml
   # Before
   args: "-no-fail -fmt sarif -out gosec-results.sarif ./..."

   # After
   args: "-fmt sarif -out gosec-results.sarif -severity high -confidence medium ./..."
   ```

2. Configure severity/confidence thresholds to fail on:
   - Severity: `high` or `critical`
   - Confidence: `medium` or `high`

3. Update CI documentation to reflect new security gate

**Benefits**:
- Catches security vulnerabilities earlier in development cycle
- Enforces higher security standards
- Aligns with enterprise best practices
- Improves overall security posture

**Timeline**:
- Target completion: After security audit phase (TBD)
- Estimated effort: 2-4 hours (prerequisite work) + 30 minutes (implementation)

---

## Other TODOs

(Add future items here)
