// Package naming holds the names and labels the InstanceType operator uses.
//
// The values are a contract with forgeai, which selects the instance types
// and the Kueue ResourceFlavors by these labels and names. Changing one breaks
// existing clusters.
package naming

import (
	"regexp"
	"strings"
)

// Labels and annotations.
const (
	// LabelManagedBy marks the objects the operator owns.
	LabelManagedBy = "app.kubernetes.io/managed-by"
	// ManagedBy is the LabelManagedBy value of forgeai-owned objects.
	ManagedBy = "forgeai"

	// LabelGPUClass is set on generated InstanceTypes and ResourceFlavors.
	LabelGPUClass = "forgeai.wutz.dev/gpu-class"

	// Node labels from NVIDIA GPU feature discovery.
	LabelGPUProduct = "nvidia.com/gpu.product"
	LabelGPUMemory  = "nvidia.com/gpu.memory"
)

// Keys of unstructured object maps.
const (
	KeyMetadata = "metadata"
	KeyName     = "name"
	KeyLabels   = "labels"
	KeySpec     = "spec"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// GPUClass normalises a nvidia.com/gpu.product value (NVIDIA-GeForce-RTX-5090)
// to a short DNS label (rtx-5090).
func GPUClass(product string) string {
	s := strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(product), "-"), "-")
	for _, p := range []string{"nvidia-", "geforce-", "tesla-"} {
		s = strings.TrimPrefix(s, p)
	}
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	return s
}

// InstanceTypeName is the InstanceType of count GPUs of a class, e.g. rtx-5090.x2.
func InstanceTypeName(class string, count int32) string {
	return class + ".x" + itoa(count)
}

// ResourceFlavor is the Kueue ResourceFlavor of a GPU class.
func ResourceFlavor(class string) string { return "forgeai-" + class }

func itoa(n int32) string {
	if n == 0 {
		return "0"
	}
	var b [11]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
