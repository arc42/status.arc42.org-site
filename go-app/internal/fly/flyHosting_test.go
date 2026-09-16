package fly

import "testing"

// TestDeploymentID pins what the footer may show: the deployment tag fly.io
// puts into FLY_IMAGE_REF, and nothing at all when the service does not run
// there (locally) or when the reference has an unexpected shape.
func TestDeploymentID(t *testing.T) {
	cases := map[string]struct {
		imageRef string
		want     string
	}{
		"deployed on fly": {
			"registry.fly.io/arc42-stats:deployment-01M2MMR6YW3JYJ9G10MBDRXAYA",
			"01M2MMR6YW3JYJ9G10MBDRXAYA",
		},
		"not on fly":              {"", ""},
		"tag is not a deployment": {"registry.fly.io/arc42-stats:latest", ""},
		"no tag at all":           {"registry.fly.io/arc42-stats", ""},
		"empty deployment tag":    {"registry.fly.io/arc42-stats:deployment-", ""},
		"digest instead of a tag": {"registry.fly.io/arc42-stats@sha256:abc123", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("FLY_IMAGE_REF", c.imageRef)
			if got := DeploymentID(); got != c.want {
				t.Errorf("DeploymentID() = %q, want %q", got, c.want)
			}
		})
	}
}
