package configx

// WithVault adds HashiCorp Vault as a configuration source.
// Requires the github.com/hashicorp/vault/api package in your application.
//
// Usage sketch:
//
//	cfg, err := configx.Load(
//	    configx.WithVault(configx.VaultConfig{
//	        Address: "https://vault.example.com",
//	        Token:   os.Getenv("VAULT_TOKEN"),
//	        Path:    "secret/data/myapp",
//	    }),
//	)
func WithVault(_ VaultConfig) Option {
	return func(_ *loader) {
		// Vault integration requires github.com/hashicorp/vault/api.
		// Import that package in your application and call loader with the
		// resolved secrets map via WithDefaults.
		panic("configx: WithVault is a stub — integrate vault/api in your application layer")
	}
}

// VaultConfig holds connection parameters for a HashiCorp Vault source.
type VaultConfig struct {
	Address   string
	Token     string
	Path      string
	MountPath string // defaults to "secret"
}

// WithConsul adds HashiCorp Consul KV as a configuration source.
// Requires the github.com/hashicorp/consul/api package in your application.
//
// Usage sketch:
//
//	cfg, err := configx.Load(
//	    configx.WithConsul(configx.ConsulConfig{
//	        Address: "localhost:8500",
//	        Prefix:  "myapp/",
//	    }),
//	)
func WithConsul(_ ConsulConfig) Option {
	return func(_ *loader) {
		panic("configx: WithConsul is a stub — integrate consul/api in your application layer")
	}
}

// ConsulConfig holds connection parameters for a Consul KV source.
type ConsulConfig struct {
	Address string
	Prefix  string
	Token   string
}

// WithAWSParameterStore adds AWS Systems Manager Parameter Store as a source.
// Requires the github.com/aws/aws-sdk-go-v2/service/ssm package in your application.
//
// Usage sketch:
//
//	cfg, err := configx.Load(
//	    configx.WithAWSParameterStore(configx.AWSParameterStoreConfig{
//	        Region: "ap-southeast-1",
//	        Path:   "/myapp/prod/",
//	    }),
//	)
func WithAWSParameterStore(_ AWSParameterStoreConfig) Option {
	return func(_ *loader) {
		panic("configx: WithAWSParameterStore is a stub — integrate aws-sdk-go-v2/ssm in your application layer")
	}
}

// AWSParameterStoreConfig holds parameters for an AWS SSM Parameter Store source.
type AWSParameterStoreConfig struct {
	Region  string
	Path    string // SSM path prefix, e.g. "/myapp/prod/"
	Profile string // AWS profile name (optional)
}
