package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestConfig holds configuration for tests
type TestConfig struct {
	TestRepo string
	TestTag  string
}

// getTestConfig returns test configuration from environment or defaults
func getTestConfig() TestConfig {
	repo := os.Getenv("LABEL_MOD_TEST_REPO")
	if repo == "" {
		repo = "localhost:5000/test/labeltest"
	}

	tag := os.Getenv("LABEL_MOD_TEST_TAG")
	if tag == "" {
		tag = "latest"
	}

	return TestConfig{
		TestRepo: repo,
		TestTag:  tag,
	}
}

// runCommand executes a command and returns the result
func runCommand(args ...string) (string, error) {
	cmd := exec.Command("./bin/label-mod", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// parseJSONResult parses the JSON output from label-mod
func parseJSONResult(output string) (Result, error) {
	var result Result
	err := json.Unmarshal([]byte(output), &result)
	return result, err
}

// ensureTestImage ensures we have a test image with known labels
func ensureTestImage(t *testing.T, config TestConfig) string {
	imageRef := fmt.Sprintf("%s:%s", config.TestRepo, config.TestTag)

	// First, test if the image exists
	output, err := runCommand("test", imageRef)
	if err == nil {
		// Image exists, check if it has the labels we need
		result, err := parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse test result: %v", err)
		}

		if result.Success && len(result.Current) > 0 {
			// Image exists and has labels, use it
			return imageRef
		}
	}

	// Image doesn't exist or doesn't have labels, we need to create one
	// For local registry, we'll use the default image
	fallbackImage := "localhost:5000/test/labeltest:latest"

	output, err = runCommand("test", fallbackImage)
	if err != nil {
		t.Skipf("No test image available, skipping test: %v", err)
		return ""
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse test result: %v", err)
	}

	if !result.Success {
		t.Skipf("Test image not available: %s", result.Error)
		return ""
	}

	return fallbackImage
}

func TestLabelModTestCommand(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	output, err := runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Test command failed: %v", err)
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if !result.Success {
		t.Fatalf("Test command returned error: %s", result.Error)
	}

	if result.ImageRef != imageRef {
		t.Errorf("Expected image_ref %s, got %s", imageRef, result.ImageRef)
	}

	if result.NewDigest == "" {
		t.Error("Expected new_digest to be set")
	}

	if result.Current == nil {
		t.Error("Expected current labels to be set")
	}

	t.Logf("Test image has %d labels", len(result.Current))
}

func TestLabelModRemoveLabels(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	// First, get current state
	output, err := runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Failed to get current state: %v", err)
	}

	initialResult, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse initial state: %v", err)
	}

	if !initialResult.Success {
		t.Fatalf("Failed to get initial state: %s", initialResult.Error)
	}

	// Find a label to remove
	var labelToRemove string
	for label := range initialResult.Current {
		labelToRemove = label
		break
	}

	if labelToRemove == "" {
		t.Skip("No labels available to remove")
		return
	}

	// Remove the label
	output, err = runCommand("remove-labels", imageRef, labelToRemove)
	if err != nil {
		t.Fatalf("Remove labels command failed: %v", err)
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse remove result: %v", err)
	}

	if !result.Success {
		t.Fatalf("Remove labels failed: %s", result.Error)
	}

	if result.OldDigest != initialResult.NewDigest {
		t.Errorf("Expected old_digest %s, got %s", initialResult.NewDigest, result.OldDigest)
	}

	if result.NewDigest == result.OldDigest {
		t.Error("Expected new digest to be different from old digest")
	}

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed label, got %d", len(result.Removed))
	}

	if result.Removed[0] != labelToRemove {
		t.Errorf("Expected removed label %s, got %s", labelToRemove, result.Removed[0])
	}

	// Verify the label was actually removed
	output, err = runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Failed to verify removal: %v", err)
	}

	verifyResult, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse verification result: %v", err)
	}

	if !verifyResult.Success {
		t.Fatalf("Verification failed: %s", verifyResult.Error)
	}

	if verifyResult.NewDigest != result.NewDigest {
		t.Errorf("Expected verification digest %s, got %s", result.NewDigest, verifyResult.NewDigest)
	}

	if _, exists := verifyResult.Current[labelToRemove]; exists {
		t.Errorf("Label %s should have been removed but still exists", labelToRemove)
	}
}

func TestLabelModUpdateLabels(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	// First, get current state
	output, err := runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Failed to get current state: %v", err)
	}

	initialResult, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse initial state: %v", err)
	}

	if !initialResult.Success {
		t.Fatalf("Failed to get initial state: %s", initialResult.Error)
	}

	// Update a label
	testKey := fmt.Sprintf("test.update.label.%d", time.Now().Unix())
	testValue := fmt.Sprintf("updated-%d", time.Now().Unix())

	output, err = runCommand("update-labels", imageRef, fmt.Sprintf("%s=%s", testKey, testValue))
	if err != nil {
		t.Fatalf("Update labels command failed: %v", err)
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse update result: %v", err)
	}

	if !result.Success {
		t.Fatalf("Update labels failed: %s", result.Error)
	}

	if result.OldDigest != initialResult.NewDigest {
		t.Errorf("Expected old_digest %s, got %s", initialResult.NewDigest, result.OldDigest)
	}

	if result.NewDigest == result.OldDigest {
		t.Error("Expected new digest to be different from old digest")
	}

	if len(result.Updated) != 1 {
		t.Errorf("Expected 1 updated label, got %d", len(result.Updated))
	}

	if result.Updated[testKey] != testValue {
		t.Errorf("Expected updated value %s, got %s", testValue, result.Updated[testKey])
	}

	// Verify the label was actually updated
	output, err = runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	verifyResult, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse verification result: %v", err)
	}

	if !verifyResult.Success {
		t.Fatalf("Verification failed: %s", verifyResult.Error)
	}

	if verifyResult.NewDigest != result.NewDigest {
		t.Errorf("Expected verification digest %s, got %s", result.NewDigest, verifyResult.NewDigest)
	}

	if verifyResult.Current[testKey] != testValue {
		t.Errorf("Expected label value %s, got %s", testValue, verifyResult.Current[testKey])
	}
}

func TestLabelModModifyLabels(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	// First, get current state
	output, err := runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Failed to get current state: %v", err)
	}

	initialResult, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse initial state: %v", err)
	}

	if !initialResult.Success {
		t.Fatalf("Failed to get initial state: %s", initialResult.Error)
	}

	// Find a label to remove
	var labelToRemove string
	for label := range initialResult.Current {
		labelToRemove = label
		break
	}

	if labelToRemove == "" {
		t.Skip("No labels available to remove")
		return
	}

	// Combined operation: remove one label and update another
	testKey := fmt.Sprintf("test.modify.label.%d", time.Now().Unix())
	testValue := fmt.Sprintf("modified-%d", time.Now().Unix())

	output, err = runCommand("modify-labels", imageRef, "--remove", labelToRemove, "--update", fmt.Sprintf("%s=%s", testKey, testValue))
	if err != nil {
		t.Fatalf("Modify labels command failed: %v", err)
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse modify result: %v", err)
	}

	if !result.Success {
		t.Fatalf("Modify labels failed: %s", result.Error)
	}

	if result.OldDigest != initialResult.NewDigest {
		t.Errorf("Expected old_digest %s, got %s", initialResult.NewDigest, result.OldDigest)
	}

	if result.NewDigest == result.OldDigest {
		t.Error("Expected new digest to be different from old digest")
	}

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed label, got %d", len(result.Removed))
	}

	if result.Removed[0] != labelToRemove {
		t.Errorf("Expected removed label %s, got %s", labelToRemove, result.Removed[0])
	}

	if len(result.Updated) != 1 {
		t.Errorf("Expected 1 updated label, got %d", len(result.Updated))
	}

	if result.Updated[testKey] != testValue {
		t.Errorf("Expected updated value %s, got %s", testValue, result.Updated[testKey])
	}

	// Verify the changes
	output, err = runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Failed to verify modify: %v", err)
	}

	verifyResult, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse verification result: %v", err)
	}

	if !verifyResult.Success {
		t.Fatalf("Verification failed: %s", verifyResult.Error)
	}

	if verifyResult.NewDigest != result.NewDigest {
		t.Errorf("Expected verification digest %s, got %s", result.NewDigest, verifyResult.NewDigest)
	}

	if _, exists := verifyResult.Current[labelToRemove]; exists {
		t.Errorf("Label %s should have been removed but still exists", labelToRemove)
	}

	if verifyResult.Current[testKey] != testValue {
		t.Errorf("Expected label value %s, got %s", testValue, verifyResult.Current[testKey])
	}
}

func TestLabelModWithTagging(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	testTag := fmt.Sprintf("test-tag-%d", time.Now().Unix())
	testKey := fmt.Sprintf("test.tag.label.%d", time.Now().Unix())
	testValue := "tagged-value"

	// Update with tagging
	output, err := runCommand("update-labels", imageRef, fmt.Sprintf("%s=%s", testKey, testValue), "--tag", testTag)
	if err != nil {
		t.Fatalf("Update with tagging command failed: %v", err)
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse update with tagging result: %v", err)
	}

	if !result.Success {
		t.Fatalf("Update with tagging failed: %s", result.Error)
	}

	if len(result.TaggedAs) == 0 {
		t.Error("Expected tagged_as to be set")
	}

	expectedTag := fmt.Sprintf("%s:%s", config.TestRepo, testTag)
	if len(result.TaggedAs) != 1 || result.TaggedAs[0] != expectedTag {
		t.Errorf("Expected tagged_as %s, got %v", expectedTag, result.TaggedAs)
	}

	// Verify the tagged image
	output, err = runCommand("test", result.TaggedAs[0])
	if err != nil {
		t.Fatalf("Failed to verify tagged image: %v", err)
	}

	verifyResult, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse tagged image result: %v", err)
	}

	if !verifyResult.Success {
		t.Fatalf("Tagged image verification failed: %s", verifyResult.Error)
	}

	if verifyResult.NewDigest != result.NewDigest {
		t.Errorf("Expected tagged image digest %s, got %s", result.NewDigest, verifyResult.NewDigest)
	}

	if verifyResult.Current[testKey] != testValue {
		t.Errorf("Expected tagged image label value %s, got %s", testValue, verifyResult.Current[testKey])
	}
}

func TestLabelModMultipleTags(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	testKey := fmt.Sprintf("test.multi.tag.%d", time.Now().Unix())
	testValue := "multi-tag-value"
	tag1 := fmt.Sprintf("multi-tag1-%d", time.Now().Unix())
	tag2 := fmt.Sprintf("multi-tag2-%d", time.Now().Unix())
	tag3 := fmt.Sprintf("multi-tag3-%d", time.Now().Unix())

	// Update with multiple tags
	output, err := runCommand("update-labels", imageRef, fmt.Sprintf("%s=%s", testKey, testValue), "--tag", tag1, "--tag", tag2, "--tag", tag3)
	if err != nil {
		t.Fatalf("Update with multiple tags command failed: %v", err)
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse update with multiple tags result: %v", err)
	}

	if !result.Success {
		t.Fatalf("Update with multiple tags failed: %s", result.Error)
	}

	if len(result.TaggedAs) != 3 {
		t.Errorf("Expected 3 tagged images, got %d", len(result.TaggedAs))
	}

	expectedTags := []string{
		fmt.Sprintf("%s:%s", config.TestRepo, tag1),
		fmt.Sprintf("%s:%s", config.TestRepo, tag2),
		fmt.Sprintf("%s:%s", config.TestRepo, tag3),
	}

	for i, expectedTag := range expectedTags {
		if i >= len(result.TaggedAs) {
			t.Errorf("Expected tag %s at position %d, but not found", expectedTag, i)
			continue
		}
		if result.TaggedAs[i] != expectedTag {
			t.Errorf("Expected tagged_as[%d] %s, got %s", i, expectedTag, result.TaggedAs[i])
		}
	}

	// Verify all tagged images
	for i, taggedImage := range result.TaggedAs {
		output, err := runCommand("test", taggedImage)
		if err != nil {
			t.Fatalf("Failed to verify tagged image %d: %v", i, err)
		}

		verifyResult, err := parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse tagged image %d result: %v", i, err)
		}

		if !verifyResult.Success {
			t.Fatalf("Tagged image %d verification failed: %s", i, verifyResult.Error)
		}

		if verifyResult.NewDigest != result.NewDigest {
			t.Errorf("Expected tagged image %d digest %s, got %s", i, result.NewDigest, verifyResult.NewDigest)
		}

		if verifyResult.Current[testKey] != testValue {
			t.Errorf("Expected tagged image %d label value %s, got %s", i, testValue, verifyResult.Current[testKey])
		}
	}
}

func TestLabelModErrorHandling(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	// Test removing non-existent label
	output, err := runCommand("remove-labels", imageRef, "nonexistent-label")
	if err == nil {
		t.Error("Expected error when removing non-existent label")
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse error result: %v", err)
	}

	if result.Success {
		t.Error("Expected success to be false for non-existent label")
	}

	if result.Error == "" {
		t.Error("Expected error message for non-existent label")
	}

	if !strings.Contains(result.Error, "No labels were removed") {
		t.Errorf("Expected error about no labels removed, got: %s", result.Error)
	}
}

func TestLabelModInvalidCommands(t *testing.T) {
	// Test invalid command
	output, err := runCommand("invalid-command")
	if err == nil {
		t.Error("Expected error for invalid command")
	}

	if !strings.Contains(output, "Unknown command") {
		t.Errorf("Expected 'Unknown command' error, got: %s", output)
	}

	// Test missing arguments
	output, err = runCommand("remove-labels")
	if err == nil {
		t.Error("Expected error for missing arguments")
	}

	if !strings.Contains(output, "Usage:") {
		t.Errorf("Expected usage message, got: %s", output)
	}
}

func TestLabelModJSONOutput(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	// Test that all commands return valid JSON
	commands := []string{"test", "remove-labels", "update-labels", "modify-labels"}

	for _, cmd := range commands {
		t.Run(fmt.Sprintf("JSON_%s", cmd), func(t *testing.T) {
			var args []string
			args = append(args, cmd)

			switch cmd {
			case "test":
				args = append(args, imageRef)
			case "remove-labels":
				args = append(args, imageRef, "test-label")
			case "update-labels":
				args = append(args, imageRef, "test.key=value")
			case "modify-labels":
				args = append(args, imageRef, "--remove", "test-label", "--update", "test.key=value")
			}

			output, err := runCommand(args...)
			if err != nil {
				// Some commands may fail (like removing non-existent labels), that's OK
				// Just verify the output is valid JSON
			}

			// Try to parse as JSON
			var result Result
			err = json.Unmarshal([]byte(output), &result)
			if err != nil {
				t.Errorf("Command %s did not return valid JSON: %v", cmd, err)
			}

			// Verify required fields
			if result.ImageRef == "" {
				t.Errorf("Command %s result missing image_ref", cmd)
			}
		})
	}
}

func TestLabelModDigestReferences(t *testing.T) {
	config := getTestConfig()
	imageRef := ensureTestImage(t, config)
	if imageRef == "" {
		return
	}

	// First, get the digest of our test image
	output, err := runCommand("test", imageRef)
	if err != nil {
		t.Fatalf("Failed to test image: %v", err)
	}

	result, err := parseJSONResult(output)
	if err != nil {
		t.Fatalf("Failed to parse test result: %v", err)
	}

	if !result.Success {
		t.Fatalf("Test image not available: %s", result.Error)
	}

	// Create digest reference
	digestRef := fmt.Sprintf("%s@%s", config.TestRepo, result.NewDigest)

	t.Run("TestDigestReference", func(t *testing.T) {
		// Test that we can read from a digest reference
		output, err := runCommand("test", digestRef)
		if err != nil {
			t.Fatalf("Failed to test digest reference: %v", err)
		}

		result, err := parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse digest test result: %v", err)
		}

		if !result.Success {
			t.Fatalf("Digest reference test failed: %s", result.Error)
		}

		if result.ImageRef != digestRef {
			t.Errorf("Expected image_ref to be %s, got %s", digestRef, result.ImageRef)
		}

		// For read-only operations, the digest should be the same
		if result.NewDigest == "" {
			t.Errorf("Expected new_digest to be set for read-only operation")
		}
	})

	t.Run("TestDigestReferenceWithoutTag", func(t *testing.T) {
		// Test that we get an error when trying to modify without a tag
		uniqueLabel := fmt.Sprintf("test.digest.label.%d", time.Now().Unix())

		// First add a label to remove
		output, err := runCommand("update-labels", imageRef, fmt.Sprintf("%s=value", uniqueLabel))
		if err != nil {
			t.Fatalf("Failed to add test label: %v", err)
		}

		result, err := parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse update result: %v", err)
		}

		if !result.Success {
			t.Fatalf("Failed to add test label: %s", result.Error)
		}

		// Now try to remove it using digest reference without tag
		output, err = runCommand("remove-labels", digestRef, uniqueLabel)
		if err == nil {
			t.Error("Expected error when removing label from digest reference without tag")
		}

		result, err = parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse error result: %v", err)
		}

		if result.Success {
			t.Error("Expected success to be false for digest reference without tag")
		}

		if !strings.Contains(result.Error, "Cannot push to digest reference without specifying a tag") {
			t.Errorf("Expected error about digest reference requiring tag, got: %s", result.Error)
		}
	})

	t.Run("TestDigestReferenceWithTag", func(t *testing.T) {
		// Test that we can modify using digest reference with a tag
		uniqueLabel := fmt.Sprintf("test.digest.tag.label.%d", time.Now().Unix())
		uniqueTag := fmt.Sprintf("digest-test-%d", time.Now().Unix())

		// First add a label to remove
		output, err := runCommand("update-labels", imageRef, fmt.Sprintf("%s=value", uniqueLabel))
		if err != nil {
			t.Fatalf("Failed to add test label: %v", err)
		}

		result, err := parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse update result: %v", err)
		}

		if !result.Success {
			t.Fatalf("Failed to add test label: %s", result.Error)
		}

		// Get the new digest
		output, err = runCommand("test", imageRef)
		if err != nil {
			t.Fatalf("Failed to get updated image digest: %v", err)
		}

		result, err = parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse test result: %v", err)
		}

		// Now remove the label using digest reference with tag
		digestRef = fmt.Sprintf("%s@%s", config.TestRepo, result.NewDigest)
		output, err = runCommand("remove-labels", digestRef, uniqueLabel, "--tag", uniqueTag)
		if err != nil {
			t.Fatalf("Failed to remove label from digest reference with tag: %v", err)
		}

		result, err = parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse remove result: %v", err)
		}

		if !result.Success {
			t.Fatalf("Failed to remove label from digest reference: %s", result.Error)
		}

		if result.ImageRef != digestRef {
			t.Errorf("Expected image_ref to be %s, got %s", digestRef, result.ImageRef)
		}

		if len(result.TaggedAs) != 1 {
			t.Errorf("Expected 1 tagged image, got %d", len(result.TaggedAs))
		}

		expectedTag := fmt.Sprintf("%s:%s", config.TestRepo, uniqueTag)
		if result.TaggedAs[0] != expectedTag {
			t.Errorf("Expected tagged image to be %s, got %s", expectedTag, result.TaggedAs[0])
		}

		// Verify the label was removed
		if !contains(result.Removed, uniqueLabel) {
			t.Errorf("Expected label %s to be in removed list", uniqueLabel)
		}
	})

	t.Run("TestDigestReferenceUpdateLabels", func(t *testing.T) {
		// Test updating labels using digest reference
		uniqueLabel := fmt.Sprintf("test.digest.update.label.%d", time.Now().Unix())
		uniqueTag := fmt.Sprintf("digest-update-test-%d", time.Now().Unix())

		// Get current digest
		output, err := runCommand("test", imageRef)
		if err != nil {
			t.Fatalf("Failed to get image digest: %v", err)
		}

		result, err := parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse test result: %v", err)
		}

		digestRef = fmt.Sprintf("%s@%s", config.TestRepo, result.NewDigest)

		// Update label using digest reference with tag
		output, err = runCommand("update-labels", digestRef, fmt.Sprintf("%s=new-value", uniqueLabel), "--tag", uniqueTag)
		if err != nil {
			t.Fatalf("Failed to update label using digest reference: %v", err)
		}

		result, err = parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse update result: %v", err)
		}

		if !result.Success {
			t.Fatalf("Failed to update label using digest reference: %s", result.Error)
		}

		if len(result.TaggedAs) != 1 {
			t.Errorf("Expected 1 tagged image, got %d", len(result.TaggedAs))
		}

		expectedTag := fmt.Sprintf("%s:%s", config.TestRepo, uniqueTag)
		if result.TaggedAs[0] != expectedTag {
			t.Errorf("Expected tagged image to be %s, got %s", expectedTag, result.TaggedAs[0])
		}

		// Verify the label was updated
		if result.Updated[uniqueLabel] != "new-value" {
			t.Errorf("Expected label %s to be updated to 'new-value', got %s", uniqueLabel, result.Updated[uniqueLabel])
		}
	})

	t.Run("TestDigestReferenceModifyLabels", func(t *testing.T) {
		// Test modify-labels command using digest reference
		uniqueRemoveLabel := fmt.Sprintf("test.digest.modify.remove.%d", time.Now().Unix())
		uniqueUpdateLabel := fmt.Sprintf("test.digest.modify.update.%d", time.Now().Unix())
		uniqueTag := fmt.Sprintf("digest-modify-test-%d", time.Now().Unix())

		// First add labels to work with
		output, err := runCommand("update-labels", imageRef,
			fmt.Sprintf("%s=old-value", uniqueRemoveLabel),
			fmt.Sprintf("%s=old-value", uniqueUpdateLabel))
		if err != nil {
			t.Fatalf("Failed to add test labels: %v", err)
		}

		result, err := parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse update result: %v", err)
		}

		if !result.Success {
			t.Fatalf("Failed to add test labels: %s", result.Error)
		}

		// Get current digest
		output, err = runCommand("test", imageRef)
		if err != nil {
			t.Fatalf("Failed to get image digest: %v", err)
		}

		result, err = parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse test result: %v", err)
		}

		digestRef = fmt.Sprintf("%s@%s", config.TestRepo, result.NewDigest)

		// Modify labels using digest reference
		output, err = runCommand("modify-labels", digestRef,
			"--remove", uniqueRemoveLabel,
			"--update", fmt.Sprintf("%s=new-value", uniqueUpdateLabel),
			"--tag", uniqueTag)
		if err != nil {
			t.Fatalf("Failed to modify labels using digest reference: %v", err)
		}

		result, err = parseJSONResult(output)
		if err != nil {
			t.Fatalf("Failed to parse modify result: %v", err)
		}

		if !result.Success {
			t.Fatalf("Failed to modify labels using digest reference: %s", result.Error)
		}

		if len(result.TaggedAs) != 1 {
			t.Errorf("Expected 1 tagged image, got %d", len(result.TaggedAs))
		}

		expectedTag := fmt.Sprintf("%s:%s", config.TestRepo, uniqueTag)
		if result.TaggedAs[0] != expectedTag {
			t.Errorf("Expected tagged image to be %s, got %s", expectedTag, result.TaggedAs[0])
		}

		// Verify the labels were modified
		if !contains(result.Removed, uniqueRemoveLabel) {
			t.Errorf("Expected label %s to be in removed list", uniqueRemoveLabel)
		}

		if result.Updated[uniqueUpdateLabel] != "new-value" {
			t.Errorf("Expected label %s to be updated to 'new-value', got %s", uniqueUpdateLabel, result.Updated[uniqueUpdateLabel])
		}
	})
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
