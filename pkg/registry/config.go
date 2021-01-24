package registry

// Config type stores the Authentication type and Artifactory URL for the upstream/staging artifactory
type Config struct {
	Hostname *string    `yaml:"hostname"`
	Auth     *BasicHttpAuth `yaml:"auth"`
	Staging bool `yaml:"staging,omitempty"`
}

// BasicHttpAuth stores the credentials for authentication with upstream/staging artifactory
type BasicHttpAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// NewBasicAuth is constructor init for HTTP Basic Auth type
func NewBasicAuth(usr, pwd string) *BasicHttpAuth {
	return &BasicHttpAuth{
		Username: usr,
		Password: pwd,
	}
}