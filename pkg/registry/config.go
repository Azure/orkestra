package registry

// Config struct captures the configuration fields as per the repoAddOptions - https://github.com/helm/helm/blob/v3.1.2/cmd/helm/repo_add.go#L39
type Config struct {
	Name               string `yaml:"name" json:"name,omitempty"`
	URL                string `yaml:"url" json:"url,omitempty"`
	Username           string `yaml:"username" json:"username,omitempty"`
	Password           string `yaml:"password" json:"password,omitempty"`
	AuthHeader         string `yaml:"authHeader" json:"auth_header,omitempty"`
	CaFile             string `yaml:"caFile" json:"ca_file,omitempty"`
	CertFile           string `yaml:"certFile" json:"cert_file,omitempty"`
	KeyFile            string `yaml:"keyFile" json:"key_file,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify" json:"insecure_skip_verify,omitempty"`
	AccessToken        string `yaml:"accessToken" json:"access_token,omitempty"`
}
