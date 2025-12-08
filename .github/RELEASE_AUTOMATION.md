# Release Automation

This document describes the automated release process for tplan.

## Overview

The GitHub Actions workflow in `.github/workflows/release.yml` automatically:
1. Builds binaries for multiple platforms
2. Creates compressed archives with checksums
3. Generates a GitHub Release with all artifacts

## Quick Start

To create a new release:

```bash
# 1. Update CHANGELOG.md with new version
vim CHANGELOG.md

# 2. Commit changes
git add CHANGELOG.md
git commit -m "Prepare release v1.0.0"
git push

# 3. Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

That's it! GitHub Actions will handle the rest.

## What Gets Built

### Platforms
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

### Artifacts
For each platform, the workflow creates:
- Binary file (e.g., `tplan-linux-amd64`)
- Compressed archive (`.tar.gz` for Unix, `.zip` for Windows)
- SHA256 checksum file (`.sha256`)

### Example Release Assets
```
tplan-linux-amd64.tar.gz
tplan-linux-amd64.tar.gz.sha256
tplan-linux-arm64.tar.gz
tplan-linux-arm64.tar.gz.sha256
tplan-darwin-amd64.tar.gz
tplan-darwin-amd64.tar.gz.sha256
tplan-darwin-arm64.tar.gz
tplan-darwin-arm64.tar.gz.sha256
tplan-windows-amd64.exe.zip
tplan-windows-amd64.exe.zip.sha256
tplan-windows-arm64.exe.zip
tplan-windows-arm64.exe.zip.sha256
```

## Workflow Details

### Trigger Conditions
- **Automatic**: When you push a tag starting with `v` (e.g., `v1.0.0`)
- **Manual**: Via GitHub Actions UI (workflow_dispatch)

### Build Process

1. **Checkout Code**: Fetches the tagged version
2. **Setup Go**: Installs Go 1.21 with caching
3. **Download Dependencies**: Runs `go mod download`
4. **Build Binary**: Compiles with optimizations:
   - `CGO_ENABLED=0` - Static binary (no C dependencies)
   - `-ldflags="-s -w"` - Strip debug info (smaller size)
   - Version embedded from git tag
5. **Compress**: Creates `.tar.gz` (Unix) or `.zip` (Windows)
6. **Checksum**: Generates SHA256 for verification
7. **Upload Artifacts**: Stores in GitHub (7-day retention)
8. **Create Release**: Publishes with auto-generated notes

### Build Optimizations

The workflow uses several optimizations:
- **Static Linking** (`CGO_ENABLED=0`): No runtime dependencies
- **Symbol Stripping** (`-s -w`): Reduces binary size by ~30%
- **Go Cache**: Speeds up builds by caching modules
- **Matrix Strategy**: Builds all platforms in parallel

### Release Notes

Each release includes:
- Feature highlights
- Installation instructions for each platform
- Usage examples
- Links to checksums
- Auto-generated commit history

## Testing the Workflow

### Before Creating a Real Release

Test the workflow locally using [act](https://github.com/nektos/act):

```bash
# Install act
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Test the workflow
act -j build --matrix os:ubuntu-latest --matrix goos:linux --matrix goarch:amd64
```

### After Creating a Release

1. Check the Actions tab for workflow status
2. Verify all jobs succeeded (green checkmarks)
3. Download and test a binary:
   ```bash
   wget https://github.com/yourusername/tplan/releases/download/v1.0.0/tplan-linux-amd64.tar.gz
   tar -xzf tplan-linux-amd64.tar.gz
   ./tplan-linux-amd64 -help
   ```

## Troubleshooting

### Build Fails

**Check the logs:**
1. Go to Actions tab
2. Click the failed workflow
3. Expand the failed step

**Common issues:**
- **Dependencies not found**: Run `go mod tidy` and commit
- **Syntax errors**: Test build locally first: `go build ./cmd/tplan`
- **Go version mismatch**: Update workflow to match your `go.mod`

### Release Not Created

**Verify:**
- Tag starts with 'v': `git tag | grep v1.0.0`
- Tag was pushed: `git ls-remote --tags origin | grep v1.0.0`
- Permissions: Check repository Settings → Actions → Workflow permissions

### Binary Size Too Large

The binaries are optimized but you can reduce further:

1. Use UPX compression (add to workflow):
   ```yaml
   - name: Compress with UPX
     run: upx --best --lzma ${{ matrix.output }}
   ```

2. Review dependencies:
   ```bash
   go list -m all
   ```

## Workflow Configuration

### Changing Build Targets

Edit `.github/workflows/release.yml`:

```yaml
strategy:
  matrix:
    include:
      # Add new platform
      - goos: freebsd
        goarch: amd64
        output: tplan-freebsd-amd64
```

### Changing Go Version

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.22'  # Update here
```

### Adding Pre-Release Support

The workflow already marks non-standard versions as pre-releases:

```bash
git tag -a v1.0.0-beta.1 -m "Beta 1"
git push origin v1.0.0-beta.1
```

This will create a pre-release on GitHub.

## Security

### Checksums

Each release includes SHA256 checksums. Users should verify:

```bash
# Download both files
wget https://github.com/yourusername/tplan/releases/download/v1.0.0/tplan-linux-amd64.tar.gz
wget https://github.com/yourusername/tplan/releases/download/v1.0.0/tplan-linux-amd64.tar.gz.sha256

# Verify
sha256sum -c tplan-linux-amd64.tar.gz.sha256
```

### Secrets

The workflow uses `GITHUB_TOKEN` which is automatically provided by GitHub Actions. No manual secret configuration needed.

## Best Practices

1. **Always update CHANGELOG.md** before releasing
2. **Test locally** before creating a tag
3. **Use semantic versioning** (v1.0.0, not 1.0.0)
4. **Write good commit messages** (they appear in release notes)
5. **Tag annotated commits** (`git tag -a`, not `git tag`)

## Maintenance

### Updating Dependencies

When updating Go or action versions:

1. Update `go.mod`: `go get -u && go mod tidy`
2. Update workflow: Edit `.github/workflows/release.yml`
3. Test locally: `go build ./cmd/tplan`
4. Create a test release: `git tag v1.0.1-test && git push origin v1.0.1-test`
5. Verify build succeeds
6. Delete test release and tag if successful

### Monitoring

Check workflow runs regularly:
- Success rate
- Build times
- Artifact sizes

Set up notifications:
- Settings → Notifications → Actions

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [actions/checkout](https://github.com/actions/checkout)
- [actions/setup-go](https://github.com/actions/setup-go)
- [softprops/action-gh-release](https://github.com/softprops/action-gh-release)
