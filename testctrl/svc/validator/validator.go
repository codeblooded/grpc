package validator

import (
	"fmt"
	"strings"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// Validator verifies that sessions conform to a list of requirements. Each of
// its fields can be used to enable, disable or adjust the requirements.
type Validator struct {
	// ImageNamePrefix enforces that all container image names have its
	// prefix. If not specified, all container image names will be valid.
	//
	// On certain registries, requiring a prefix can make the cluster more
	// secure. For example, Google Container Registry scopes container
	// images by Google Cloud Project. It assigns all images a name like
	// `gcr.io/<project>/<image>`. To enforce that all images came from a
	// specific GCR project, we can set this value to `gcr.io/<project>/`.
	ImageNamePrefix string
}

// Validate checks that the session meets all requirements. If not, it returns
// an error with the first violation it encounters.
func (v *Validator) Validate(session *types.Session) error {
	// TODO: Add more validations
	return v.validateImages(session)
}

func (v *Validator) validImageName(name string) bool {
	return strings.HasPrefix(name, v.ImageNamePrefix)
}

func (v *Validator) validateImages(session *types.Session) error {
	driver := session.Driver
	if driver == nil {
		return fmt.Errorf("driver component required, but is missing")
	}

	driverImage := driver.ContainerImage
	if !v.validImageName(driverImage) {
		return fmt.Errorf("driver container image %q does not have required prefix %q",
			driverImage, v.ImageNamePrefix)
	}

	for _, worker := range session.Workers {
		workerImage := worker.ContainerImage
		if !v.validImageName(workerImage) {
			return fmt.Errorf("%v container image %q does not have required prefix %q",
				strings.ToLower(worker.Kind.String()), workerImage, v.ImageNamePrefix)
		}
	}
	return nil
}
