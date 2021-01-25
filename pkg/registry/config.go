package registry

// Config type stores the Authentication type and Artifactory URL for the upstream/staging artifactory
type Config struct {
	Hostname *string        `yaml:"hostname"`
	Auth     *BasicHTTPAuth `yaml:"auth"`
	Staging  bool           `yaml:"staging,omitempty"`
}

// BasicHttpAuth stores the credentials for authentication with upstream/staging artifactory
type BasicHTTPAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// NewBasicAuth is constructor init for HTTP Basic Auth type
func NewBasicAuth(usr, pwd string) *BasicHTTPAuth {
	return &BasicHTTPAuth{
		Username: usr,
		Password: pwd,
	}
}
