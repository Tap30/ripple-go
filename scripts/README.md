# Scripts

## sync-version.sh

Synchronizes the version from `.versionrc` to `version.go`.

**Usage:**
```bash
./scripts/sync-version.sh
```

**What it does:**
1. Reads version from `.versionrc`
2. Validates semantic version format (x.x.x or x.x.x-suffix)
3. Updates `version.go` with the new version constant
4. Reports the changes

**Makefile integration:**
```bash
make version-sync    # Run the sync script
make version-check   # Verify versions are consistent
```
