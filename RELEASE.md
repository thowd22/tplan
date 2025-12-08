# Release Guide

This document explains how to create a new release of tplan.

## Automated Release Process

The project uses GitHub Actions to automatically build binaries and create releases when you push a version tag.

### Creating a Release

1. **Update the version number**

   Update `CHANGELOG.md` with the new version and changes:
   ```bash
   # Edit CHANGELOG.md and add the new version section
   vim CHANGELOG.md
   ```

2. **Commit your changes**
   ```bash
   git add CHANGELOG.md
   git commit -m "Prepare release v1.0.0"
   git push origin main
   ```

3. **Create and push a version tag**
   ```bash
   # Create an annotated tag
   git tag -a v1.0.0 -m "Release v1.0.0"
   
   # Push the tag to GitHub
   git push origin v1.0.0
   ```

4. **GitHub Actions automatically:**
   - Builds binaries for all platforms (Linux, macOS, Windows)
   - Builds for both amd64 and arm64 architectures
   - Creates compressed archives (.tar.gz for Unix, .zip for Windows)
   - Generates SHA256 checksums for each archive
   - Creates a GitHub Release with all artifacts
   - Generates release notes with installation instructions

### Supported Platforms

The release workflow builds binaries for:

- **Linux**
  - `tplan-linux-amd64` (64-bit Intel/AMD)
  - `tplan-linux-arm64` (64-bit ARM)

- **macOS**
  - `tplan-darwin-amd64` (Intel Macs)
  - `tplan-darwin-arm64` (Apple Silicon)

- **Windows**
  - `tplan-windows-amd64.exe` (64-bit Intel/AMD)
  - `tplan-windows-arm64.exe` (64-bit ARM)

### Manual Trigger

You can also manually trigger the release workflow from the GitHub Actions tab:

1. Go to the **Actions** tab in GitHub
2. Select **Build and Release** workflow
3. Click **Run workflow**
4. Select the branch and click **Run workflow**

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v2.0.0): Incompatible API changes
- **MINOR** version (v1.1.0): New functionality (backward compatible)
- **PATCH** version (v1.0.1): Bug fixes (backward compatible)

Examples:
- `v1.0.0` - First major release
- `v1.1.0` - Added new feature
- `v1.0.1` - Fixed a bug
- `v2.0.0` - Breaking changes

### Pre-releases

For beta or release candidate versions, use:

```bash
git tag -a v1.0.0-beta.1 -m "Beta release v1.0.0-beta.1"
git push origin v1.0.0-beta.1
```

The workflow will mark these as "pre-release" on GitHub.

## Release Checklist

Before creating a release:

- [ ] All tests pass (`go test ./...`)
- [ ] Binary builds successfully (`go build ./cmd/tplan`)
- [ ] CHANGELOG.md is updated with new version
- [ ] README.md reflects current features
- [ ] Documentation is up to date
- [ ] No open critical bugs
- [ ] Version tag follows semantic versioning

## Testing the Release

After creating a release:

1. **Download the binary for your platform**
   ```bash
   wget https://github.com/yourusername/tplan/releases/download/v1.0.0/tplan-linux-amd64.tar.gz
   ```

2. **Verify the checksum**
   ```bash
   wget https://github.com/yourusername/tplan/releases/download/v1.0.0/tplan-linux-amd64.tar.gz.sha256
   sha256sum -c tplan-linux-amd64.tar.gz.sha256
   ```

3. **Extract and test**
   ```bash
   tar -xzf tplan-linux-amd64.tar.gz
   ./tplan-linux-amd64 -help
   ```

4. **Test with real Terraform plan**
   ```bash
   terraform plan | ./tplan-linux-amd64
   ```

## Hotfix Releases

For urgent bug fixes:

1. Create a hotfix branch from the release tag:
   ```bash
   git checkout -b hotfix/v1.0.1 v1.0.0
   ```

2. Make your fixes and commit:
   ```bash
   git add .
   git commit -m "Fix critical bug"
   ```

3. Update CHANGELOG.md with the hotfix version

4. Merge to main and create tag:
   ```bash
   git checkout main
   git merge hotfix/v1.0.1
   git tag -a v1.0.1 -m "Hotfix v1.0.1"
   git push origin main
   git push origin v1.0.1
   ```

## Rollback

If a release has issues:

1. **Delete the tag locally and remotely:**
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

2. **Delete the GitHub Release:**
   - Go to the Releases page
   - Click on the release
   - Click "Delete this release"

3. **Fix the issues and create a new patch release**

## GitHub Actions Workflow

The workflow file is located at `.github/workflows/release.yml`.

### Workflow Features:

- **Trigger:** Runs on version tags (v*)
- **Matrix Build:** Builds for multiple OS/arch combinations in parallel
- **Optimization:** Uses `-ldflags="-s -w"` to reduce binary size
- **Versioning:** Embeds version from git tag into binary
- **Checksums:** Generates SHA256 for verification
- **Release Notes:** Auto-generated with installation instructions
- **Artifacts:** Stores built binaries for 7 days

### Customizing the Workflow

To modify the platforms or build settings:

1. Edit `.github/workflows/release.yml`
2. Update the `matrix.include` section to add/remove platforms
3. Modify `ldflags` in the build step to change build options
4. Update release notes template in the workflow

## Troubleshooting

### Build Fails

Check the GitHub Actions logs:
1. Go to Actions tab
2. Click on the failed workflow run
3. Check the build logs for errors

Common issues:
- Go version mismatch
- Missing dependencies (run `go mod tidy`)
- Syntax errors in code

### Release Not Created

Ensure:
- Tag starts with 'v' (e.g., v1.0.0, not 1.0.0)
- Tag is pushed to GitHub (`git push origin v1.0.0`)
- GitHub Actions has write permissions to contents

### Binary Doesn't Work

Check:
- Correct platform (don't use Linux binary on macOS)
- File has execute permissions (`chmod +x tplan`)
- Dependencies are met (shouldn't be any for CGO_ENABLED=0 builds)

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
