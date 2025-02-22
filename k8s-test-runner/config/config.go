package config

import (
	"encoding/base64"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Runner struct {
	Namespace               string            `toml:"namespace" validate:"required"`
	RBACRoleName            string            `toml:"rbac_role_name" validate:"required"`
	SyncValue               string            `toml:"sync_value"` // Random if not set
	ImageRegistryURL        string            `toml:"image_registry_url" envconfig:"IMAGE_REGISTRY_URL" validate:"required"`
	ImageName               string            `toml:"image_name" validate:"required"`
	ImageTag                string            `toml:"image_tag" validate:"required"`
	TestName                string            `toml:"test_name" validate:"required"`
	TestConfigBase64EnvName string            `toml:"test_config_base64_env_name"`
	TestConfigFilePath      string            `toml:"test_config_file_path" envconfig:"TEST_CONFIG_FILE_PATH"`
	TestConfigBase64        string            `toml:"test_config_base64" envconfig:"TEST_CONFIG_BASE64"`
	TestTimeout             time.Duration     `toml:"test_timeout" validate:"required"`
	ResourcesRequestsCPU    string            `toml:"resources_requests_cpu" validate:"required"`    // Eg. "1000m"
	ResourcesRequestsMemory string            `toml:"resources_requests_memory" validate:"required"` // Eg. "512Mi"
	ResourcesLimitsCPU      string            `toml:"resources_limits_cpu" validate:"required"`      // Eg. "1000m"
	ResourcesLimitsMemory   string            `toml:"resources_limits_memory" validate:"required"`   // Eg. "512Mi"
	JobCount                int               `toml:"job_count" validate:"required"`
	KeepJobs                bool              `toml:"keep_jobs"`
	UpdateImage             bool              `toml:"update_image"`
	ChartPath               string            `toml:"chart_path"` // Default chart if not set
	Envs                    map[string]string `toml:"envs"`
}

// Read initializes the configuration by sequentially loading from a TOML file,
// a base64 encoded string, and finally from environment variables. Each step
// potentially overwrites previously set values, allowing for flexible configuration
// precedence. This function is particularly useful for handling configuration in
// different environments like local development, CI, or production.
//
// Parameters:
// - tomlFilePath: File path to a TOML configuration file.
// - base64Config: Base64 encoded configuration string.
// - targetConfig: Pointer to the struct where the configuration will be stored.
//
// The function first tries to read from the TOML file if 'tomlFilePath' is not empty.
// Then, it attempts to overwrite the configuration with the base64 encoded string
// provided in 'base64Config'. Finally, it overwrites any existing configuration with
// values from environment variables.
//
// Returns an error if any step of reading and unmarshaling configurations fails.
func Read(tomlFilePath, base64Config string, targetConfig interface{}) error {
	// Load configuration from the TOML file if a path is provided.
	if tomlFilePath != "" {
		tomlData, err := os.ReadFile(tomlFilePath)
		if err != nil {
			return errors.Wrapf(err, "error reading TOML test config file at %s", tomlFilePath)
		}
		if err := toml.Unmarshal(tomlData, targetConfig); err != nil {
			return errors.Wrap(err, "error unmarshaling TOML data for test config")
		}
	}

	// Override configuration with base64 encoded string if provided.
	if base64Config != "" {
		decodedBase64Config, err := base64.StdEncoding.DecodeString(base64Config)
		if err != nil {
			return errors.Wrap(err, "error decoding base64 config string")
		}
		if err := toml.Unmarshal(decodedBase64Config, targetConfig); err != nil {
			return errors.Wrap(err, "error unmarshaling base64 decoded data for test config")
		}
	}

	// Further override configuration with environment variables.
	if err := envconfig.Process("", targetConfig); err != nil {
		return errors.Wrap(err, "error processing environment variables for test config")
	}

	// Validate the configuration
	validate := validator.New()
	err := validate.Struct(targetConfig)

	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return errors.Wrap(err, "error validating test config")
		}

		for _, err := range err.(validator.ValidationErrors) {
			// Customize the error message based on the validation tag
			switch err.Tag() {
			case "oneof":
				return errors.Wrapf(err, "error validating test config. The field '%s' must be one of [%s].", err.Field(), err.Param())
			default:
				return errors.Wrap(err, "error validating test config")
			}
		}
	}
	return nil
}
