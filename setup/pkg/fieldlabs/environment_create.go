package fieldlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/replicatedhq/replicated/cli/print"
	"github.com/replicatedhq/replicated/pkg/kotsclient"
	"github.com/replicatedhq/replicated/pkg/types"
)

const (
	OneWeek = 7 * 24 * time.Hour
)

type Environment struct {
	// name of the environment
	Name string `json:"name,omitempty" csv:"name"`
	// slug of the environment
	Slug string `json:"slug,omitempty" csv:"slug"`
	// public key of the user who will access this environment
	PubKey string `json:"pub_key,omitempty" csv:"pub_key"`
	// email to invite to vendor.replicated.com if Params.InviteUsers is set
	Email string `json:"email,omitempty" csv:"email"`
	// password to be set on kotsadm instances
	KotsadmPassword string `json:"password,omitempty" csv:"password"`

	App types.App `json:"-" csv:"-"`
}

type LabSpec struct {
	// Name of the lab
	Name string `json:"name"`
	// Slug of the lab
	Slug string `json:"slug"`
	// Channel name to create
	Channel string `json:"channel"`
	// Slug of channel name
	ChannelSlug string `json:"channel_slug"`
	// Customer Name to Create
	Customer string `json:"customer"`
	// Dir of YAML sources to promote to channel
	YAMLDir string `json:"yaml_dir"`
	// Path to Installer YAML to promote to channel
	K8sInstallerYAMLPath string `json:"k8s_installer_yaml_path"`

	// whether to include a license file and install KOTS
	SkipInstallKots bool `json:"skip_install_kots"`
	// whether to include a license file and install the app
	SkipInstallApp bool `json:"skip_install_app"`
	// kots config values to pass during install
	ConfigValues string `json:"config_values"`
	// bash source to run before installing KOTS
	PreInstallSH string `json:"pre_install_sh"`
	// bash source to run after installing KOTS
	PostInstallSH string `json:"post_install_sh"`

	// add a public ip?
	UsePublicIP bool `json:"use_public_ip"`
	// add a squid proxy?
	UseProxy bool `json:"use_proxy"`
	// add a jump box?
	UseJumpBox bool `json:"use_jump_box"`
}

type Token struct {
	AccessToken string `json:"access_token"`
}

type Instance struct {
	Name          string `json:"name"`
	Prefix        string `json:"prefix"`
	InstallScript string `json:"provision_sh"`
	MachineType   string `json:"machine_type"`
	BookDiskGB    string `json:"boot_disk_gb"`

	// used to define if a proxy server should be used.
	UseProxy bool `json:"use_proxy"`

	// used in a tf for_each, just put nils in both, the keys and values are ignored
	PublicIps map[string]interface{} `json:"public_ips"`
}

type Lab struct {
	Spec   LabSpec
	Status LabStatus
}

type LabStatus struct {
	InstanceToMake Instance
	Env            Environment

	App       types.App
	Channel   *types.Channel
	Customer  *types.Customer
	Release   *types.ReleaseInfo
	Installer *types.InstallerSpec
}

type EnvironmentManager struct {
	Log     *print.Logger
	Writer  io.Writer
	Params  *Params
	Client  *kotsclient.VendorV3Client
	GClient *kotsclient.GraphQLClient
}

func (e *EnvironmentManager) Validate(envs []Environment, labs []LabSpec) error {
	slug.CustomSub = map[string]string{"_": "-"}

	for _, env := range envs {
		if env.Name == "" {
			return errors.Errorf("no name set for env %v", env)
		}

		if env.Slug == "" {
			return errors.Errorf("no slug set for env %s", env.Name)
		}

		slugified := slug.Make(env.Slug)
		if env.Slug != slugified {
			return errors.Errorf("slugified form of env.Slug %q didn't match provided slug %q", slugified, env.Slug)
		}

		// slug is used as the password to all instances in the environment, so make sure it meets the kots minimum reqs
		if env.KotsadmPassword == "" {
			return errors.Errorf("no kotsadm password set for env %s", env.Name)
		}
	}

	for _, lab := range labs {
		slugifiedChannel := slug.Make(lab.Channel)

		if lab.ChannelSlug == "" {
			lab.ChannelSlug = slugifiedChannel
		}

		if slugifiedChannel != lab.ChannelSlug {
			return errors.Errorf("slugified form of Channel name %q was %q, did not match specified slug %q", lab.Channel, slugifiedChannel, lab.ChannelSlug)
		}
	}

	return nil
}
func (e *EnvironmentManager) Ensure(envs []Environment, labSpecs []LabSpec) error {
	if e.Params.InviteUsers {
		err := e.inviteUsers(envs)
		if err != nil {
			return errors.Wrap(err, "invite users")
		}
	}

	envs, appsToCreate, appsToDelete, err := e.getAppsDelta(envs)
	if err != nil {
		return errors.Wrap(err, "calculating deltas fails")
	}

	// confirm delete
	answer, err := PromptConfirm()
	if err != nil {
		return errors.Wrap(err, "confirm fails")
	}

	if answer != "yes" {
		return errors.New("prompt declined")
	}

	e.deleteApps(appsToDelete)

	envs, err = e.createApps(appsToCreate, envs)
	if err != nil {
		return errors.Wrap(err, "create apps")
	}

	labStatuses, err := e.createVendorLabs(envs, labSpecs)
	if err != nil {
		return errors.Wrap(err, "create vendor labs")
	}

	err = e.mergeWriteTFInstancesJSON(labStatuses)
	if err != nil {
		return errors.Wrap(err, "write tf instances json")
	}

	e.Log.ActionWithoutSpinner("Preparing terraform command:")
	fmt.Printf("make instances provisioner_json_out=%q\n", e.Params.InstanceJSONOutput)

	return nil
}

func (e *EnvironmentManager) getAppsDelta(envs []Environment) ([]Environment, []string, []types.App, error) {
	var outEnvs []Environment
	var appsToKeep []types.App
	var appsToCreate []string
	var appsToDelete []types.App
	apps, err := e.GClient.ListApps()
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "list apps")
	}

	for _, env := range envs {
		appName := fmt.Sprintf("%s-%s", e.Params.NamePrefix, env.Slug)
		exists := false
		for _, app := range apps {
			match, _ := regexp.MatchString(fmt.Sprintf("^%s-", appName), app.App.Slug)
			if app.App.ID == appName || app.App.Slug == appName || match {
				exists = true
				env.App = *app.App
				appsToKeep = append(appsToKeep, *app.App)
				break // App already exists and is also contained in envs
			}
		}
		if !exists {
			// Aop was not found, so will need to be created.
			appsToCreate = append(appsToCreate, appName)
		}
		outEnvs = append(outEnvs, env)
	}

	for _, app := range apps {
		if strings.Contains(app.App.Slug, e.Params.NamePrefix) && !contains(appsToKeep, *app.App) {
			appsToDelete = append(appsToDelete, *app.App)
		}
	}

	e.Log.Info("Following Apps will be created: %d", len(appsToCreate))
	for _, appName := range appsToCreate {
		e.Log.Info("%s", appName)
	}
	e.Log.Info("Following Apps will be removed: %d", len(appsToDelete))
	err = e.PrintApps(appsToDelete)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "print apps to delete")
	}
	return outEnvs, appsToCreate, appsToDelete, nil
}

func contains(apps []types.App, app types.App) bool {
	for _, v := range apps {
		if v.Slug == app.Slug {
			return true
		}
	}

	return false
}

// write TF json for the instances
// merging the new instances with any already present in the
// json file
func (e *EnvironmentManager) mergeWriteTFInstancesJSON(labStatuses []Lab) error {

	gcpInstances := map[string]Instance{}

	for _, labInstance := range labStatuses {
		if _, ok := gcpInstances[labInstance.Status.InstanceToMake.Name]; ok {
			e.Log.Error(errors.Errorf("WARNING -- instance %q already present in %q, refusing to overwrite", labInstance.Status.InstanceToMake.Name, e.Params.InstanceJSONOutput))
		}
		gcpInstances[labInstance.Status.InstanceToMake.Name] = labInstance.Status.InstanceToMake
		if labInstance.Spec.UseJumpBox {
			jumpBoxName := fmt.Sprintf("%s-jump", labInstance.Status.InstanceToMake.Name)
			gcpInstances[jumpBoxName] = Instance{
				Name:   jumpBoxName,
				Prefix: e.Params.NamePrefix,
				InstallScript: fmt.Sprintf(`
#!/bin/bash 
# add new user
sudo useradd -s /bin/bash -d /home/kots -m -p safWNrcAGYqm2 -G sudo kots
sudo groups kots
echo 'kots ALL=(ALL)        NOPASSWD: ALL' | sudo EDITOR='tee -a' visudo
# update ssh to allow password login
sudo sed -i 's/no/yes/g' /etc/ssh/sshd_config
sudo service ssh restart
# user must change password on first login
sudo chage --lastday 0 kots

set -euo pipefail

mkdir -p ~/.ssh
cat <<EOF >>~/.ssh/authorized_keys
%s
EOF
`, labInstance.Status.Env.PubKey),
				MachineType: "n1-standard-1",
				BookDiskGB:  "10",
				PublicIps: map[string]interface{}{
					"_": nil,
				},
			}
		}
	}

	serialized, err := json.MarshalIndent(gcpInstances, "", "  ")
	if err != nil {
		return errors.Wrap(err, "serialize instance json")
	}

	err = ioutil.WriteFile(e.Params.InstanceJSONOutput, serialized, 0644)
	if err != nil {
		return errors.Wrapf(err, "write file %q", e.Params.InstanceJSONOutput)
	}

	return nil
}

func (e *EnvironmentManager) createVendorLabs(envs []Environment, labSpecs []LabSpec) ([]Lab, error) {
	var labs []Lab

	for _, env := range envs {
		app := env.App
		for _, labSpec := range labSpecs {
			var lab Lab
			lab.Spec = labSpec
			lab.Status.App = app
			lab.Status.Env = env
			appLabSlug := fmt.Sprintf("%s-%s", lab.Status.App.Slug, lab.Spec.Slug)
			e.Log.ActionWithSpinner("Provision lab %s", appLabSlug)

			kotsYAML, err := readYAMLDir(labSpec.YAMLDir)
			if err != nil {
				return nil, errors.Wrapf(err, "read yaml dir %q", labSpec.YAMLDir)
			}

			kurlYAML, err := ioutil.ReadFile(labSpec.K8sInstallerYAMLPath)
			if err != nil {
				return nil, errors.Wrapf(err, "read installer yaml %q", labSpec.K8sInstallerYAMLPath)
			}

			channel, err := e.getOrCreateChannel(lab)
			if err != nil {
				return nil, errors.Wrapf(err, "get or create channel %q", lab.Spec.Channel)
			}
			lab.Status.Channel = channel

			customer, err := e.getOrCreateCustomer(lab)
			if err != nil {
				return nil, errors.Wrapf(err, "create customer for lab %q app %q", labSpec.Slug, app.Slug)
			}
			lab.Status.Customer = customer

			release, err := e.GClient.CreateRelease(app.ID, kotsYAML)
			if err != nil {
				return nil, errors.Wrapf(err, "create release for %q", labSpec.YAMLDir)
			}

			lab.Status.Release = release

			err = e.GClient.PromoteRelease(app.ID, release.Sequence, labSpec.Slug, labSpec.Name, channel.ID)
			if err != nil {
				return nil, errors.Wrapf(err, "promote release %d to channel %q", release.Sequence, channel.Slug)
			}

			installer, err := e.GClient.CreateInstaller(app.ID, string(kurlYAML))
			if err != nil {
				return nil, errors.Wrapf(err, "create installer from %q", labSpec.K8sInstallerYAMLPath)
			}
			lab.Status.Installer = installer

			err = e.GClient.PromoteInstaller(app.ID, installer.Sequence, channel.ID, labSpec.Slug)
			if err != nil {
				return nil, errors.Wrapf(err, "promote installer %d to channel %q", installer.Sequence, channel.Slug)
			}

			licenseContents, err := e.Client.DownloadLicense(app.ID, customer.ID)
			if err != nil {
				return nil, errors.Wrap(err, "download license")
			}

			kotsProvisionScript := fmt.Sprintf(`
curl -fSsL https://k8s.kurl.sh/%s-%s | sudo bash &> kURL.output
`, lab.Status.App.Slug, lab.Spec.ChannelSlug)

			appProvisioningScript := fmt.Sprintf(`
KUBECONFIG=/etc/kubernetes/admin.conf kubectl patch secret kotsadm-tls -p '{"metadata": {"annotations": {"acceptAnonymousUploads": "0"}}}'

KUBECONFIG=/etc/kubernetes/admin.conf kubectl kots install %s-%s \
	--license-file ./license.yaml \
	--namespace default \
	--shared-password %s			
`, lab.Status.App.Slug, lab.Spec.ChannelSlug, env.KotsadmPassword)

			if lab.Spec.SkipInstallKots && lab.Spec.SkipInstallApp {
				kotsProvisionScript = ""
			}

			if lab.Spec.SkipInstallApp {
				appProvisioningScript = ""
			}

			publicIPs := map[string]interface{}{}
			if lab.Spec.UsePublicIP {
				// hack: used in a tf for_each loop, just need something here
				publicIPs["_"] = nil
			}

			lab.Status.InstanceToMake = Instance{
				Name:        appLabSlug,
				Prefix:      e.Params.NamePrefix,
				MachineType: "n1-standard-4",
				BookDiskGB:  "200",
				UseProxy:    lab.Spec.UseProxy,
				PublicIps:   publicIPs,
				InstallScript: fmt.Sprintf(`
#!/bin/bash 
# add new user
sudo useradd -s /bin/bash -d /home/kots -m -p safWNrcAGYqm2 -G sudo kots
sudo groups kots
echo 'kots ALL=(ALL)        NOPASSWD: ALL' | sudo EDITOR='tee -a' visudo
# update ssh to allow password login
sudo sed -i 's/no/yes/g' /etc/ssh/sshd_config
sudo service ssh restart
# user must change password on first login
sudo chage --lastday 0 kots

set -euo pipefail

%s

cat <<EOF > ./license.yaml
%s
EOF

mkdir -p ~/.ssh
cat <<EOF >>~/.ssh/authorized_keys
# added by kots-field-labs
%s
EOF

%s

%s

%s
`, lab.Spec.PreInstallSH, licenseContents, env.PubKey, kotsProvisionScript, appProvisioningScript, lab.Spec.PostInstallSH),
			}
			e.Log.FinishSpinner()
			labs = append(labs, lab)
		}
	}

	return labs, nil
}

func (e *EnvironmentManager) getOrCreateChannel(lab Lab) (*types.Channel, error) {
	channels, err := e.Client.ListChannels(lab.Status.App.ID, lab.Status.App.Slug, lab.Spec.Channel)
	if err != nil {
		return nil, errors.Wrapf(err, "list channel %q for app %q", lab.Spec.Channel, lab.Status.App.Slug)
	}

	var matchedChannels []types.Channel
	for _, channel := range channels {
		if channel.Name == lab.Spec.Channel || channel.Slug == lab.Spec.Channel {
			matchedChannels = append(matchedChannels, channel)
		}
	}

	if len(matchedChannels) == 1 {
		return &matchedChannels[0], nil
	}

	if len(matchedChannels) > 1 {
		return nil, errors.Errorf("expected at most one channel to match %q, found %d", lab.Spec.Channel, len(matchedChannels))
	}

	channel, err := e.GClient.CreateChannel(lab.Status.App.ID, lab.Spec.Slug, lab.Spec.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "create channel for lab %q app %q", lab.Spec.Slug, lab.Status.App.Slug)
	}
	return channel, nil
}

func (e *EnvironmentManager) getOrCreateCustomer(lab Lab) (*types.Customer, error) {
	customers, err := e.Client.ListCustomers(lab.Status.App.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "list customer %q for app %q", lab.Spec.Channel, lab.Status.App.Slug)
	}

	for _, customer := range customers {
		if customer.Name == lab.Spec.Customer {
			return &customer, nil
		}
	}

	customer, err := e.Client.CreateCustomer(lab.Spec.Customer, lab.Status.App.ID, lab.Status.Channel.ID, OneWeek)
	if err != nil {
		return nil, errors.Wrapf(err, "create customer for lab %q app %q", lab.Spec.Slug, lab.Status.App.Slug)
	}
	return customer, nil
}

func (e *EnvironmentManager) createApps(appsToCreate []string, envs []Environment) ([]Environment, error) {
	var outEnvs []Environment
	var appsCreated []types.App
	for _, env := range envs {
		appName := fmt.Sprintf("%s-%s", e.Params.NamePrefix, env.Slug)
		for _, appToCreate := range appsToCreate {
			if appName == appToCreate {
				app, err := e.createApp(appName)
				if err != nil {
					return nil, errors.Wrapf(err, "create app %q", appName)
				}
				env.App = *app
				appsCreated = append(appsCreated, *app)
			}
		}
		outEnvs = append(outEnvs, env)
	}
	_ = e.PrintApps(appsCreated)
	return outEnvs, nil
}

func (e *EnvironmentManager) createApp(appName string) (*types.App, error) {
	e.Log.ActionWithSpinner(fmt.Sprintf("Creating App %s", appName))
	app, err := e.Client.CreateKOTSApp(appName)
	if err != nil {
		e.Log.FinishSpinnerWithError()
		return nil, errors.Wrapf(err, "create app %s", appName)
	}
	e.Log.FinishSpinner()

	return &types.App{
		ID:        app.ID,
		Name:      app.Name,
		Scheduler: "kots",
		Slug:      app.Slug,
	}, nil
}

func (e *EnvironmentManager) inviteUsers(envs []Environment) error {
	for _, env := range envs {
		if env.Email == "" {
			continue
		}
		inviteBody := map[string]string{
			"email":     env.Email,
			"policy_id": e.Params.RBACPolicyID,
		}
		inviteBodyBytes, err := json.Marshal(inviteBody)
		if err != nil {
			return errors.Wrap(err, "marshal invite body")
		}
		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/v1/team/invite", e.Params.IDOrigin),
			bytes.NewReader(inviteBodyBytes),
		)
		if err != nil {
			return errors.Wrap(err, "build invite request")
		}
		req.Header.Set("Authorization", e.Params.SessionToken)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "send invite request")
		}
		defer resp.Body.Close()
		// rate limit returned when already invited
		if resp.StatusCode == 429 {
			e.Log.ActionWithoutSpinner("Skipping invite %q due to 429 error", env.Email)
			continue
		}
		if resp.StatusCode != 204 {
			body, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf("POST /team/invite %d: %s", resp.StatusCode, body)
		}
	}
	return nil
}