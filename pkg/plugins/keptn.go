package plugins

import (
	"fmt"
	"net/http"
	"os"

	"code.gitea.io/sdk/gitea"
)

const (
	KeptnPluginName = "keptn"

	GitURLParam      = "url"
	GitUsernameParam = "username"
	GitPasswordParam = "password"
	GitAPITokenParam = "git-token"

	GitAdminUsername   = "gitea-admin"
	GitDefaultUsername = "keptn"
	GitDefaultPassword = "password"
)

type Git struct {
	URL      string
	Username string
	Password string

	APIToken string
}

// Keptn implements the plugin interface
type Keptn struct {
	git Git
}

func New(url string) (*Keptn, error) {
	var user, pwd string
	if user = os.Getenv("GITEA_USERNAME"); user != "" {
		user = GitDefaultUsername
	}

	if pwd = os.Getenv("GITEA_PASSWORD"); pwd != "" {
		pwd = GitDefaultPassword
	}

	return &Keptn{
		git: Git{
			URL:      url,
			Username: user,
			Password: pwd,
		},
	}, nil
}

// Init initializes the plugin by interacting with the plugin components
// For the Keptn plugin we initialize the Gitea component with a new user
// using the admin credentials
func (k *Keptn) Init() error {
	// Initialize Gitea with a new user
	if err := k.initGit(); err != nil {
		return err
	}

	// TODO: Verify that keptn API is up and running
	return nil
}

func (p *Keptn) Name() string {
	return KeptnPluginName
}

func (k *Keptn) GetParam(name string) string {
	switch name {
	case GitURLParam:
		return k.git.URL
	case GitUsernameParam:
		return k.git.Username
	case GitPasswordParam:
		return k.git.Password
	case GitAPITokenParam:
		return k.git.APIToken
	}
	return ""
}

func (k *Keptn) initGit() error {
	admin, err := k.initAdminClient()
	if err != nil {
		return err
	}

	var token string

	token, err = k.createUser(admin)
	if err != nil {
		return err
	}

	k.git.APIToken = token

	return nil
}
func (k *Keptn) initAdminClient() (*gitea.Client, error) {
	var adminUname string
	var adminPwd string

	if adminUname = os.Getenv("GITEA_ADMIN_USERNAME"); adminUname == "" {
		adminUname = GitAdminUsername
	}

	if adminPwd = os.Getenv("GITEA_ADMIN_PASSWORD"); adminPwd == "" {
		return nil, fmt.Errorf("gitea admin password is required")
	}

	admin, err := gitea.NewClient(k.git.URL, gitea.SetBasicAuth(adminUname, adminPwd))
	if err != nil {
		return nil, err
	}

	return admin, nil
}

func (k *Keptn) createUser(admin *gitea.Client) (string, error) {
	opts := gitea.CreateUserOption{
		LoginName: k.git.Username,
		Username:  k.git.Username,
		FullName:  k.git.Username,
		Password:  k.git.Password,
	}

	if _, resp, err := admin.AdminCreateUser(opts); err != nil || resp.StatusCode != http.StatusCreated {
		return "", err
	}

	uClient, err := gitea.NewClient(k.git.URL, gitea.SetBasicAuth(k.git.Username, k.git.Password))
	if err != nil {
		return "", err
	}

	t, resp, err := uClient.CreateAccessToken(gitea.CreateAccessTokenOption{
		Name: KeptnPluginName,
	})
	if err != nil || resp.StatusCode != http.StatusCreated {
		return "", err
	}

	return t.Token, nil
}
