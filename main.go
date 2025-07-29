package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Config struct {
	Registry     string
	Username     string
	Password     string
	Repository   string
	Tag          string
	NewTag       string
	RemoveLabels []string
	UpdateLabels map[string]string
}

type Result struct {
	Success   bool              `json:"success"`
	Error     string            `json:"error,omitempty"`
	ImageRef  string            `json:"image_ref"`
	OldDigest string            `json:"old_digest,omitempty"`
	NewDigest string            `json:"new_digest,omitempty"`
	Removed   []string          `json:"removed,omitempty"`
	Updated   map[string]string `json:"updated,omitempty"`
	Current   map[string]string `json:"current,omitempty"`
	TaggedAs  []string          `json:"tagged_as,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./label-mod <command>")
		fmt.Println("Commands:")
		fmt.Println("  remove-labels <image> <label1> [label2] ... [--tag <new-tag>] [--tag <another-tag>] ...")
		fmt.Println("  update-labels <image> <key=value> [key=value] ... [--tag <new-tag>] [--tag <another-tag>] ...")
		fmt.Println("  modify-labels <image> [--remove <label1>] [--remove <label2>] [--update <key=value>] [--update <key=value>] [--tag <new-tag>] [--tag <another-tag>] ...")
		fmt.Println("  test <image>")
		fmt.Println("Example:")
		fmt.Println("  ./label-mod remove-labels quay.io/bcook/labeltest/test:latest quay.expires-after")
		fmt.Println("  ./label-mod remove-labels quay.io/bcook/labeltest/test:latest quay.expires-after --tag no-expiry --tag latest")
		fmt.Println("  ./label-mod update-labels quay.io/bcook/labeltest/test:latest quay.expires-after=2024-12-31 --tag updated --tag v1.0")
		fmt.Println("  ./label-mod modify-labels quay.io/bcook/labeltest/test:latest --remove quay.expires-after --update test.label=new-value --tag modified --tag stable")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "remove-labels":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./label-mod remove-labels <image> <label1> [label2] ... [--tag <new-tag>]")
			os.Exit(1)
		}
		image := os.Args[2]
		args := os.Args[3:]
		labelsToRemove, newTags := parseArgs(args)
		result := removeLabels(image, labelsToRemove, newTags)
		outputJSON(result)

	case "update-labels":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./label-mod update-labels <image> <key=value> [key=value] ... [--tag <new-tag>]")
			os.Exit(1)
		}
		image := os.Args[2]
		args := os.Args[3:]
		labelUpdates, newTags := parseUpdateArgs(args)
		result := updateLabels(image, labelUpdates, newTags)
		outputJSON(result)

	case "modify-labels":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./label-mod modify-labels <image> [--remove <label1>] [--remove <label2>] [--update <key=value>] [--update <key=value>] [--tag <new-tag>]")
			os.Exit(1)
		}
		image := os.Args[2]
		args := os.Args[3:]
		labelsToRemove, labelUpdates, newTags := parseModifyArgs(args)
		result := modifyLabels(image, labelsToRemove, labelUpdates, newTags)
		outputJSON(result)

	case "test":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./label-mod test <image>")
			os.Exit(1)
		}
		image := os.Args[2]
		result := testImage(image)
		outputJSON(result)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func outputJSON(result Result) {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))

	if !result.Success {
		os.Exit(1)
	}
}

// tagImage handles tagging an image (always allowed)
func tagImage(ref name.Reference, newImg v1.Image, auth authn.Authenticator) error {
	return remote.Write(ref, newImg, remote.WithAuth(auth))
}

// pushImageWithDigestHandling handles pushing an image with proper digest reference handling
func pushImageWithDigestHandling(ref name.Reference, newImg v1.Image, auth authn.Authenticator, newTags []string) error {
	// Check if this is a digest reference
	if _, ok := ref.(name.Digest); ok {
		// For digest references, we can't push back to the same digest
		// We need to either tag it or create a new digest reference
		if len(newTags) == 0 {
			return fmt.Errorf("cannot push to digest reference without specifying a tag - use --tag to specify a new tag")
		}
		// Don't push to the original digest reference, only tag
		return nil
	}

	// Push the updated image to the original reference
	return remote.Write(ref, newImg, remote.WithAuth(auth))
}

func parseArgs(args []string) ([]string, []string) {
	var labelsToRemove []string
	var newTags []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--tag" && i+1 < len(args) {
			newTags = append(newTags, args[i+1])
			i++ // skip the tag value
		} else {
			labelsToRemove = append(labelsToRemove, args[i])
		}
	}

	return labelsToRemove, newTags
}

func parseUpdateArgs(args []string) (map[string]string, []string) {
	updates := make(map[string]string)
	var newTags []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--tag" && i+1 < len(args) {
			newTags = append(newTags, args[i+1])
			i++ // skip the tag value
		} else {
			parts := strings.SplitN(args[i], "=", 2)
			if len(parts) == 2 {
				updates[parts[0]] = parts[1]
			}
		}
	}

	return updates, newTags
}

func parseModifyArgs(args []string) ([]string, map[string]string, []string) {
	var labelsToRemove []string
	var labelUpdates map[string]string
	var newTags []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--remove" && i+1 < len(args) {
			labelsToRemove = append(labelsToRemove, args[i+1])
			i++ // skip the label value
		} else if args[i] == "--update" && i+1 < len(args) {
			if labelUpdates == nil {
				labelUpdates = make(map[string]string)
			}
			parts := strings.SplitN(args[i+1], "=", 2)
			if len(parts) == 2 {
				labelUpdates[parts[0]] = parts[1]
			}
			i++ // skip the update value
		} else if args[i] == "--tag" && i+1 < len(args) {
			newTags = append(newTags, args[i+1])
			i++ // skip the tag value
		}
	}

	return labelsToRemove, labelUpdates, newTags
}

func removeLabels(imageRef string, labelsToRemove []string, newTags []string) Result {
	result := Result{
		ImageRef: imageRef,
		Removed:  []string{},
	}

	// Parse image reference
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		result.Error = fmt.Sprintf("Error parsing image reference: %v", err)
		return result
	}

	// Get authentication using go-containerregistry
	auth, err := authn.DefaultKeychain.Resolve(ref.Context())
	if err != nil {
		result.Error = fmt.Sprintf("Error getting authentication: %v", err)
		return result
	}

	// Get image using go-containerregistry
	img, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		result.Error = fmt.Sprintf("Error getting image: %v", err)
		return result
	}

	// Get old digest
	oldDigest, err := img.Digest()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting old digest: %v", err)
		return result
	}
	result.OldDigest = oldDigest.String()

	// Get config using go-containerregistry
	config, err := img.ConfigFile()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting config: %v", err)
		return result
	}

	// Check if this is a digest reference before attempting to modify
	if _, ok := ref.(name.Digest); ok {
		// For digest references, we can't push back to the same digest
		// We need to either tag it or create a new digest reference
		if len(newTags) == 0 {
			result.Error = "Cannot push to digest reference without specifying a tag. Use --tag to specify a new tag."
			return result
		}
	}

	// Remove labels
	removed := false
	for _, label := range labelsToRemove {
		if _, exists := config.Config.Labels[label]; exists {
			delete(config.Config.Labels, label)
			result.Removed = append(result.Removed, label)
			removed = true
		}
	}

	if !removed {
		result.Error = "No labels were removed"
		return result
	}

	// Create new image with updated config
	newImg, err := mutate.Config(img, config.Config)
	if err != nil {
		result.Error = fmt.Sprintf("Error updating config: %v", err)
		return result
	}

	// Push the updated image
	if err := pushImageWithDigestHandling(ref, newImg, auth, newTags); err != nil {
		result.Error = fmt.Sprintf("Error pushing updated image: %v", err)
		return result
	}

	// Get the digest of the new image
	digest, err := newImg.Digest()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting digest: %v", err)
		return result
	}
	result.NewDigest = digest.String()
	result.Success = true

	// If new tags were specified, tag the image
	if len(newTags) > 0 {
		result.TaggedAs = make([]string, 0, len(newTags))
		for _, tag := range newTags {
			newRef, err := name.NewTag(fmt.Sprintf("%s:%s", ref.Context().String(), tag))
			if err != nil {
				result.Error = fmt.Sprintf("Error creating new tag reference: %v", err)
				return result
			}

			if err := tagImage(newRef, newImg, auth); err != nil {
				result.Error = fmt.Sprintf("Error tagging image: %v", err)
				return result
			}

			result.TaggedAs = append(result.TaggedAs, newRef.String())
		}
	}

	return result
}

func updateLabels(imageRef string, labelUpdates map[string]string, newTags []string) Result {
	result := Result{
		ImageRef: imageRef,
		Updated:  make(map[string]string),
	}

	// Parse image reference
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		result.Error = fmt.Sprintf("Error parsing image reference: %v", err)
		return result
	}

	// Get authentication using go-containerregistry
	auth, err := authn.DefaultKeychain.Resolve(ref.Context())
	if err != nil {
		result.Error = fmt.Sprintf("Error getting authentication: %v", err)
		return result
	}

	// Get image using go-containerregistry
	img, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		result.Error = fmt.Sprintf("Error getting image: %v", err)
		return result
	}

	// Get old digest
	oldDigest, err := img.Digest()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting old digest: %v", err)
		return result
	}
	result.OldDigest = oldDigest.String()

	// Get config using go-containerregistry
	config, err := img.ConfigFile()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting config: %v", err)
		return result
	}

	// Check if this is a digest reference before attempting to modify
	if _, ok := ref.(name.Digest); ok {
		// For digest references, we can't push back to the same digest
		// We need to either tag it or create a new digest reference
		if len(newTags) == 0 {
			result.Error = "Cannot push to digest reference without specifying a tag. Use --tag to specify a new tag."
			return result
		}
	}

	// Update labels
	if config.Config.Labels == nil {
		config.Config.Labels = make(map[string]string)
	}

	for key, value := range labelUpdates {
		config.Config.Labels[key] = value
		result.Updated[key] = value
	}

	// Create new image with updated config
	newImg, err := mutate.Config(img, config.Config)
	if err != nil {
		result.Error = fmt.Sprintf("Error updating config: %v", err)
		return result
	}

	// Push the updated image
	if err := pushImageWithDigestHandling(ref, newImg, auth, newTags); err != nil {
		result.Error = fmt.Sprintf("Error pushing updated image: %v", err)
		return result
	}

	// Get the digest of the new image
	digest, err := newImg.Digest()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting digest: %v", err)
		return result
	}
	result.NewDigest = digest.String()
	result.Success = true

	// If new tags were specified, tag the image
	if len(newTags) > 0 {
		result.TaggedAs = make([]string, 0, len(newTags))
		for _, tag := range newTags {
			newRef, err := name.NewTag(fmt.Sprintf("%s:%s", ref.Context().String(), tag))
			if err != nil {
				result.Error = fmt.Sprintf("Error creating new tag reference: %v", err)
				return result
			}

			if err := tagImage(newRef, newImg, auth); err != nil {
				result.Error = fmt.Sprintf("Error tagging image: %v", err)
				return result
			}

			result.TaggedAs = append(result.TaggedAs, newRef.String())
		}
	}

	return result
}

func modifyLabels(imageRef string, labelsToRemove []string, labelUpdates map[string]string, newTags []string) Result {
	result := Result{
		ImageRef: imageRef,
		Removed:  []string{},
		Updated:  make(map[string]string),
	}

	// Parse image reference
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		result.Error = fmt.Sprintf("Error parsing image reference: %v", err)
		return result
	}

	// Get authentication using go-containerregistry
	auth, err := authn.DefaultKeychain.Resolve(ref.Context())
	if err != nil {
		result.Error = fmt.Sprintf("Error getting authentication: %v", err)
		return result
	}

	// Get image using go-containerregistry
	img, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		result.Error = fmt.Sprintf("Error getting image: %v", err)
		return result
	}

	// Get old digest
	oldDigest, err := img.Digest()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting old digest: %v", err)
		return result
	}
	result.OldDigest = oldDigest.String()

	// Get config using go-containerregistry
	config, err := img.ConfigFile()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting config: %v", err)
		return result
	}

	// Check if this is a digest reference before attempting to modify
	if _, ok := ref.(name.Digest); ok {
		// For digest references, we can't push back to the same digest
		// We need to either tag it or create a new digest reference
		if len(newTags) == 0 {
			result.Error = "Cannot push to digest reference without specifying a tag. Use --tag to specify a new tag."
			return result
		}
	}

	// Remove labels
	for _, label := range labelsToRemove {
		if _, exists := config.Config.Labels[label]; exists {
			delete(config.Config.Labels, label)
			result.Removed = append(result.Removed, label)
		}
	}

	// Update labels
	if config.Config.Labels == nil {
		config.Config.Labels = make(map[string]string)
	}

	for key, value := range labelUpdates {
		config.Config.Labels[key] = value
		result.Updated[key] = value
	}

	// Create new image with updated config
	newImg, err := mutate.Config(img, config.Config)
	if err != nil {
		result.Error = fmt.Sprintf("Error updating config: %v", err)
		return result
	}

	// Push the updated image
	if err := pushImageWithDigestHandling(ref, newImg, auth, newTags); err != nil {
		result.Error = fmt.Sprintf("Error pushing updated image: %v", err)
		return result
	}

	// Get the digest of the new image
	digest, err := newImg.Digest()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting digest: %v", err)
		return result
	}
	result.NewDigest = digest.String()
	result.Success = true

	// If new tags were specified, tag the image
	if len(newTags) > 0 {
		result.TaggedAs = make([]string, 0, len(newTags))
		for _, tag := range newTags {
			newRef, err := name.NewTag(fmt.Sprintf("%s:%s", ref.Context().String(), tag))
			if err != nil {
				result.Error = fmt.Sprintf("Error creating new tag reference: %v", err)
				return result
			}

			if err := tagImage(newRef, newImg, auth); err != nil {
				result.Error = fmt.Sprintf("Error tagging image: %v", err)
				return result
			}

			result.TaggedAs = append(result.TaggedAs, newRef.String())
		}
	}

	return result
}

func testImage(imageRef string) Result {
	result := Result{
		ImageRef: imageRef,
		Current:  make(map[string]string),
	}

	// Parse image reference
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		result.Error = fmt.Sprintf("Error parsing image reference: %v", err)
		return result
	}

	// Get authentication using go-containerregistry
	auth, err := authn.DefaultKeychain.Resolve(ref.Context())
	if err != nil {
		result.Error = fmt.Sprintf("Error getting authentication: %v", err)
		return result
	}

	// Get image using go-containerregistry
	img, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		result.Error = fmt.Sprintf("Error getting image: %v", err)
		return result
	}

	// Get config using go-containerregistry
	config, err := img.ConfigFile()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting config: %v", err)
		return result
	}

	// Get the digest
	digest, err := img.Digest()
	if err != nil {
		result.Error = fmt.Sprintf("Error getting digest: %v", err)
		return result
	}
	result.NewDigest = digest.String()
	result.Current = config.Config.Labels
	result.Success = true

	return result
}
