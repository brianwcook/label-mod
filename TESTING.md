# Testing Guide for label-mod

This document describes how to run and configure tests for the `label-mod` tool.

## Overview

The test suite includes comprehensive tests for all `label-mod` functionality:

- ✅ **Test command** - Verifies image inspection and label reading
- ✅ **Remove labels** - Tests label removal with verification
- ✅ **Update labels** - Tests label updates with verification  
- ✅ **Modify labels** - Tests combined remove/update operations
- ✅ **Tagging** - Tests image tagging functionality
- ✅ **Error handling** - Tests error cases and edge conditions
- ✅ **JSON output** - Verifies all commands return valid JSON
- ✅ **Invalid commands** - Tests command validation

## Running Tests

### Basic Test Run
```bash
go test -v
```

### Run Specific Test
```bash
go test -v -run TestLabelModTestCommand
go test -v -run TestLabelModRemoveLabels
go test -v -run TestLabelModUpdateLabels
go test -v -run TestLabelModModifyLabels
go test -v -run TestLabelModWithTagging
go test -v -run TestLabelModErrorHandling
```

### Run All Tests with Coverage
```bash
go test -v -cover
```

## Configuration

### Environment Variables

The test suite is configurable via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `LABEL_MOD_TEST_REPO` | `quay.io/bcook/labeltest/test` | Test repository |
| `LABEL_MOD_TEST_TAG` | `test-{timestamp}` | Test image tag |

### Example Configurations

**Use default test image:**
```bash
go test -v
```

**Use specific test image:**
```bash
LABEL_MOD_TEST_REPO=quay.io/bcook/labeltest/test LABEL_MOD_TEST_TAG=has-label go test -v
```

**Use your own test repository:**
```bash
LABEL_MOD_TEST_REPO=quay.io/your-repo/test-image LABEL_MOD_TEST_TAG=latest go test -v
```

## Test Requirements

### Authentication
Tests assume authentication is already configured (e.g., via `docker login` or `podman login`).

### Test Image Requirements
- Must be accessible via the configured repository/tag
- Should have at least one label for removal tests
- Should be writable for update/modify tests

### Fallback Behavior
If the configured test image is not available, tests will:
1. Try to use `quay.io/bcook/labeltest/test:has-label` as fallback
2. Skip tests if no suitable image is available

## Test Structure

### Test Functions

| Test Function | Purpose |
|---------------|---------|
| `TestLabelModTestCommand` | Tests the `test` command functionality |
| `TestLabelModRemoveLabels` | Tests label removal with verification |
| `TestLabelModUpdateLabels` | Tests label updates with verification |
| `TestLabelModModifyLabels` | Tests combined operations |
| `TestLabelModWithTagging` | Tests tagging functionality |
| `TestLabelModErrorHandling` | Tests error cases |
| `TestLabelModInvalidCommands` | Tests command validation |
| `TestLabelModJSONOutput` | Verifies JSON output format |

### Helper Functions

| Function | Purpose |
|----------|---------|
| `getTestConfig()` | Returns test configuration from environment |
| `runCommand(args...)` | Executes label-mod command |
| `parseJSONResult(output)` | Parses JSON output from label-mod |
| `ensureTestImage(t, config)` | Ensures test image is available |

## Test Output

### Successful Test
```
=== RUN   TestLabelModTestCommand
    label_mod_test.go:126: Test image has 2 labels
--- PASS: TestLabelModTestCommand (2.47s)
```

### Skipped Test (no image available)
```
=== RUN   TestLabelModTestCommand
--- SKIP: TestLabelModTestCommand (0.00s)
    label_mod_test.go:89: No test image available, skipping test: exit status 1
```

### Failed Test
```
=== RUN   TestLabelModTestCommand
--- FAIL: TestLabelModTestCommand (0.02s)
    label_mod_test.go:115: Test command returned error: Error getting authentication
```

## Continuous Integration

### GitHub Actions Example
```yaml
name: Test label-mod
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - name: Login to Quay
        run: |
          echo ${{ secrets.QUAY_PASSWORD }} | docker login quay.io -u ${{ secrets.QUAY_USERNAME }} --password-stdin
      - name: Run tests
        run: |
          go build -o label-mod main.go
          go test -v
        env:
          LABEL_MOD_TEST_REPO: quay.io/your-repo/test-image
          LABEL_MOD_TEST_TAG: latest
```

## Troubleshooting

### Common Issues

**Authentication Errors:**
```
Error getting authentication: no authentication available
```
- Ensure you're logged in to the registry: `docker login quay.io`
- Or set up credentials via environment variables

**Image Not Found:**
```
No test image available, skipping test
```
- Verify the test image exists in the configured repository
- Check that the image has labels for testing

**Permission Errors:**
```
Error pushing updated image: UNAUTHORIZED
```
- Ensure you have push permissions to the test repository
- Use a repository you own for testing

### Debug Mode

To see detailed command execution:
```bash
go test -v -run TestLabelModTestCommand -test.v
```

## Test Coverage

The test suite covers:

- ✅ **All commands** (test, remove-labels, update-labels, modify-labels)
- ✅ **Success cases** with verification
- ✅ **Error cases** with proper error messages
- ✅ **JSON output** validation
- ✅ **Digest tracking** (old vs new)
- ✅ **Tagging functionality**
- ✅ **Edge cases** (no labels, invalid commands)

## Contributing

When adding new tests:

1. Follow the existing naming convention: `TestLabelMod{Function}`
2. Use the helper functions for consistency
3. Include proper error checking and verification
4. Add appropriate test documentation
5. Ensure tests are configurable via environment variables 