# GitHub Wiki Setup Instructions

This document explains how to populate the LuminetIQ GitHub Wiki with the prepared Wiki content.

## Overview

All Wiki pages have been pre-created in `docs/wiki/` directory. These files need to be uploaded to the GitHub Wiki, which is a separate Git repository.

## Quick Setup (Recommended)

### 1. Enable GitHub Wiki

1. Go to https://github.com/krisarmstrong/luminetiq/settings
2. Scroll to "Features" section
3. Check ✅ "Wikis"
4. Save changes

### 2. Clone the Wiki Repository

```bash
# Navigate to project root
cd /Users/krisarmstrong/Developer/projects/netscope

# Clone the wiki (it's a separate git repo)
git clone https://github.com/krisarmstrong/luminetiq.wiki.git wiki-repo

# Copy prepared pages
cp docs/wiki/*.md wiki-repo/

# Commit and push
cd wiki-repo
git add .
git commit -m "Initial wiki setup: Hardware compatibility documentation"
git push origin master

# Cleanup
cd ..
rm -rf wiki-repo
```

### 3. Verify

Visit https://github.com/krisarmstrong/luminetiq/wiki to see the populated wiki.

---

## Manual Setup (Alternative)

If you prefer to create pages via the web interface:

### 1. Create Home Page

1. Go to https://github.com/krisarmstrong/luminetiq/wiki
2. Click "Create the first page"
3. Copy content from `docs/wiki/Home.md`
4. Paste into editor
5. Click "Save Page"

### 2. Create Vendor Pages

For each vendor page:

1. Click "New Page" in the wiki
2. Set page title (exactly as shown):
   - **Wi-Fi:** `Intel-WiFi`, `Qualcomm-Atheros-WiFi`, `Broadcom-WiFi`, `Realtek-WiFi`, `MediaTek-WiFi`
   - **Ethernet:** `Intel-Ethernet`, `Broadcom-Ethernet`, `Realtek-Ethernet`, `Marvell-Ethernet`

3. Copy content from corresponding file in `docs/wiki/`
4. Paste and save

### Page Title Naming Rules

⚠️ **Important:** GitHub Wiki converts page titles to filenames

- Use hyphens for spaces: `Intel WiFi` → `Intel-WiFi`
- Case-sensitive: Must match exactly as referenced in links
- No special characters

**Correct:**
- `Intel-WiFi.md` → Links work ✅
- `Qualcomm-Atheros-WiFi.md` → Links work ✅

**Incorrect:**
- `Intel WiFi.md` → Links break ❌
- `intel-wifi.md` → Links break ❌

---

## Wiki Page Checklist

After setup, verify all pages exist:

### Wi-Fi Adapters (5 pages)
- [ ] Home.md
- [ ] Intel-WiFi.md
- [ ] Qualcomm-Atheros-WiFi.md
- [ ] Broadcom-WiFi.md
- [ ] Realtek-WiFi.md
- [ ] MediaTek-WiFi.md

### Ethernet NICs (4 pages)
- [ ] Intel-Ethernet.md
- [ ] Broadcom-Ethernet.md
- [ ] Realtek-Ethernet.md
- [ ] Marvell-Ethernet.md

**Total:** 10 pages (1 home + 5 Wi-Fi + 4 Ethernet)

---

## Linking Pages

All pages use relative links that work both locally and on GitHub Wiki:

### Link Format
```markdown
[Link Text](Page-Name)
```

### Examples
```markdown
[Intel Wi-Fi Adapters](Intel-WiFi)
[Back to Home](Home)
[Next: Broadcom →](Broadcom-WiFi)
```

### Link Verification

After publishing, click through all navigation links to verify:
- Home page links to all vendor pages ✅
- Each vendor page links back to Home ✅
- Sequential navigation (Previous/Next) works ✅

---

## Updating the Wiki

### Regular Updates

The Wiki should be updated:
- Quarterly (per HARDWARE_DOCUMENTATION_PLAN.md)
- When new chipsets are released
- When community submits hardware reports
- When driver support improves

### Update Process

**Option 1: Clone, edit, push (recommended for bulk updates)**
```bash
git clone https://github.com/krisarmstrong/luminetiq.wiki.git
cd luminetiq.wiki
# Edit files
git add .
git commit -m "Update: Description of changes"
git push origin master
```

**Option 2: Edit via web (quick single-page updates)**
1. Navigate to page on GitHub
2. Click "Edit" button
3. Make changes
4. Save with commit message

### Keeping Local Copy in Sync

After updating via web interface:
```bash
# Update local docs/wiki/ to match
cd /Users/krisarmstrong/Developer/projects/netscope
git clone https://github.com/krisarmstrong/luminetiq.wiki.git /tmp/wiki
cp /tmp/wiki/*.md docs/wiki/
rm -rf /tmp/wiki

# Commit to main repo
git add docs/wiki/
git commit -m "docs: Sync wiki pages from GitHub Wiki"
git push origin main
```

---

## Community Contributions

### Enabling Community Edits

**Option 1: Open Wiki (anyone can edit)**
- Go to Repository Settings → Features
- Wiki is enabled by default for anyone to edit

**Option 2: Controlled Contributions (recommended)**
1. Disable direct wiki edits (Settings → Features)
2. Users submit hardware reports via GitHub Issues
3. Maintainers review and update wiki

### Adding Community Reports

When community submits a hardware report:

1. **Verify report completeness**
   - Check all required fields filled
   - Verify kernel/driver versions
   - Ensure testing was done properly

2. **Add to appropriate vendor page**
   ```markdown
   ### Tested Configurations

   #### Intel AX200 on Ubuntu 22.04 LTS
   **Reported by:** @username | **Date:** 2025-12-14

   - **Kernel:** 5.15.0-91-generic
   - **Driver:** iwlwifi
   - **Monitor Mode:** ✅ Excellent
   - **Channel Switching:** ✅ Fast (<500ms)
   - **Signal Quality:** ✅ Accurate
   - **Notes:** Works perfectly out of box, no configuration needed.
   ```

3. **Update compatibility matrix** (if new info)

4. **Thank the contributor**
   - Comment on their issue
   - Link to the wiki page update
   - Close issue with "Added to wiki" label

---

## Wiki Maintenance Schedule

### Weekly (First 3 months)
- Check for new hardware report issues
- Add verified reports to wiki
- Monitor for broken links

### Monthly
- Review popular pages for accuracy
- Update pricing if significantly changed
- Check for new chipset announcements

### Quarterly (Ongoing)
- Full review per HARDWARE_DOCUMENTATION_PLAN.md
- Update kernel version recommendations
- Deprecate EOL hardware
- Verify all external links

---

## Troubleshooting

### Problem: Links Not Working

**Cause:** Page title doesn't match link

**Solution:**
```bash
# Check exact page title in wiki
# Ensure links match exactly (case-sensitive)

# Example fix:
# Link: [Intel WiFi](Intel-WiFi)  ← Must match page name exactly
# Page name: Intel-WiFi.md         ← Verify this matches
```

### Problem: Formatting Issues

**Cause:** Markdown rendering differences

**Solution:**
- Preview locally first using `glow` or similar
  ```bash
  glow docs/wiki/Home.md
  ```
- GitHub Wiki uses GitHub-Flavored Markdown (GFM)
- Test in a scratch wiki first if unsure

### Problem: Images Not Displaying

**Note:** Current pages don't use images, but if added:

**Solution:**
```markdown
# Upload images to wiki repo
# Reference with relative path
![Alt text](images/screenshot.png)

# Or use external URL
![Alt text](https://example.com/image.png)
```

---

## Integration with Main Documentation

### Cross-Linking

**From README.md → Wiki:**
```markdown
See the [Hardware Compatibility Wiki](https://github.com/krisarmstrong/luminetiq/wiki) for detailed compatibility reports.
```

**From HARDWARE.md → Wiki:**
```markdown
Community-tested configurations: [Wiki - Intel Wi-Fi](https://github.com/krisarmstrong/luminetiq/wiki/Intel-WiFi)
```

**From Wiki → Main Docs:**
```markdown
Official hardware guide: [HARDWARE.md](https://github.com/krisarmstrong/luminetiq/blob/main/HARDWARE.md)
```

---

## Success Metrics

Track Wiki effectiveness:

### Engagement Metrics
- Page views (GitHub provides basic stats)
- Number of community reports submitted
- External links to wiki pages

### Quality Metrics
- <2% broken links
- Updated within last 90 days
- >5 community reports per quarter

### User Satisfaction
- Reduced "hardware doesn't work" issues
- Positive feedback in issues/discussions
- External citations (Reddit, forums)

---

## Next Steps

After wiki is populated:

1. **Announce in README.md**
   ```markdown
   ## 📋 Hardware Compatibility

   Check the [Wiki](https://github.com/krisarmstrong/luminetiq/wiki) for community-tested hardware compatibility reports.
   ```

2. **Add link to issue template**
   - Update `.github/ISSUE_TEMPLATE/hardware-report.yml`
   - Add note: "Your report will be added to the [Wiki](link)"

3. **Promote in community channels**
   - Reddit post
   - Discord announcement
   - Twitter/X post

4. **Recruit initial testers**
   - Ask 3-5 users to test reference hardware
   - Seed wiki with quality reports

---

**Created:** 2025-12-14
**Maintained by:** LuminetIQ Documentation Team
