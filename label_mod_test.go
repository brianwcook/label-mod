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
		repo = "quay.io/bcook/labeltest/test"
	}

	tag := os.Getenv("LABEL_MOD_TEST_TAG")
	if tag == "" {
		tag = fmt.Sprintf("test-%d", time.Now().Unix())
	}

	return TestConfig{
		TestRepo: repo,
		TestTag:  tag,
	}
}

// runCommand executes a command and returns the result
func runCommand(args ...string) (string, error) {
	cmd := exec.Command("./label-mod", args...)
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
	// For now, we'll use a known image that should exist
	fallbackImage := "quay.io/bcook/labeltest/test:has-label"

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

	if result.TaggedAs == "" {
		t.Error("Expected tagged_as to be set")
	}

	expectedTag := fmt.Sprintf("%s:%s", strings.Split(imageRef, ":")[0], testTag)
	if result.TaggedAs != expectedTag {
		t.Errorf("Expected tagged_as %s, got %s", expectedTag, result.TaggedAs)
	}

	// Verify the tagged image
	output, err = runCommand("test", result.TaggedAs)
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
