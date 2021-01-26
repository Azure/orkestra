package registry

// Config type stores the Authentication type and Artifactory URL for the upstream/staging artifactory
// type Config struct {
// 	Hostname *string        `yaml:"hostname"`
// 	Auth     *BasicHTTPAuth `yaml:"auth"`
// 	Staging  bool           `yaml:"staging,omitempty"`
// }

type Config struct {
	Name               string
	URL                string `yaml:"url" json:"url,omitempty"`
	Username           string `yaml:"username" json:"username,omitempty"`
	Password           string `yaml:"password" json:"password,omitempty"`
	AccessToken        string `yaml:"accessToken" json:"access_token,omitempty"`
	AuthHeader         string `yaml:"authHeader" json:"auth_header,omitempty"`
	CaFile             string `yaml:"caFile" json:"ca_file,omitempty"`
	CertFile           string `yaml:"certFile" json:"cert_file,omitempty"`
	KeyFile            string `yaml:"keyFile" json:"key_file,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify" json:"insecure_skip_verify,omitempty"`
}
