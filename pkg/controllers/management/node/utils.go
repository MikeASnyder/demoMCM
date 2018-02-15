package node

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var regExHyphen = regexp.MustCompile("([a-z])([A-Z])")

var (
	RegExNodeDirEnv      = regexp.MustCompile("^" + nodeDirEnvKey + ".*")
	RegExNodePluginToken = regexp.MustCompile("^" + "MACHINE_PLUGIN_TOKEN=" + ".*")
	RegExNodeDriverName  = regexp.MustCompile("^" + "MACHINE_PLUGIN_DRIVER_NAME=" + ".*")
)

const (
	errorCreatingNode = "Error creating machine: "
	nodeDirEnvKey     = "MACHINE_STORAGE_PATH="
	nodeCmd           = "docker-machine"
)

func buildCreateCommand(node *v3.Node, configMap map[string]interface{}) []string {
	sDriver := strings.ToLower(node.Status.NodeTemplateSpec.Driver)
	cmd := []string{"create", "-d", sDriver}

	cmd = append(cmd, buildEngineOpts("--engine-install-url", []string{node.Status.NodeTemplateSpec.EngineInstallURL})...)
	cmd = append(cmd, buildEngineOpts("--engine-opt", mapToSlice(node.Status.NodeTemplateSpec.EngineOpt))...)
	cmd = append(cmd, buildEngineOpts("--engine-env", mapToSlice(node.Status.NodeTemplateSpec.EngineEnv))...)
	cmd = append(cmd, buildEngineOpts("--engine-insecure-registry", node.Status.NodeTemplateSpec.EngineInsecureRegistry)...)
	cmd = append(cmd, buildEngineOpts("--engine-label", mapToSlice(node.Status.NodeTemplateSpec.EngineLabel))...)
	cmd = append(cmd, buildEngineOpts("--engine-registry-mirror", node.Status.NodeTemplateSpec.EngineRegistryMirror)...)
	cmd = append(cmd, buildEngineOpts("--engine-storage-driver", []string{node.Status.NodeTemplateSpec.EngineStorageDriver})...)

	for k, v := range configMap {
		dmField := "--" + sDriver + "-" + strings.ToLower(regExHyphen.ReplaceAllString(k, "${1}-${2}"))
		switch v.(type) {
		case int64:
			cmd = append(cmd, dmField, strconv.FormatInt(v.(int64), 10))
		case string:
			if v.(string) != "" {
				cmd = append(cmd, dmField, v.(string))
			}
		case bool:
			if v.(bool) {
				cmd = append(cmd, dmField)
			}
		case []interface{}:
			for _, s := range v.([]interface{}) {
				if _, ok := s.(string); ok {
					cmd = append(cmd, dmField, s.(string))
				}
			}
		}
	}
	logrus.Debugf("create cmd %v", cmd)
	cmd = append(cmd, node.Spec.RequestedHostname)
	return cmd
}

func buildEngineOpts(name string, values []string) []string {
	var opts []string
	for _, value := range values {
		if value == "" {
			continue
		}
		opts = append(opts, name, value)
	}
	return opts
}

func mapToSlice(m map[string]string) []string {
	var ret []string
	for k, v := range m {
		ret = append(ret, fmt.Sprintf("%s=%s", k, v))
	}
	return ret
}

func buildCommand(nodeDir string, cmdArgs []string) *exec.Cmd {
	command := exec.Command(nodeCmd, cmdArgs...)
	env := initEnviron(nodeDir)
	command.Env = env
	return command
}

func initEnviron(nodeDir string) []string {
	env := os.Environ()
	found := false
	for idx, ev := range env {
		if RegExNodeDirEnv.MatchString(ev) {
			env[idx] = nodeDirEnvKey + nodeDir
			found = true
		}
		if RegExNodePluginToken.MatchString(ev) {
			env[idx] = ""
		}
		if RegExNodeDriverName.MatchString(ev) {
			env[idx] = ""
		}
	}
	if !found {
		env = append(env, nodeDirEnvKey+nodeDir)
	}
	return env
}

func startReturnOutput(command *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	readerStdout, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	readerStderr, err := command.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := command.Start(); err != nil {
		readerStdout.Close()
		readerStderr.Close()
		return nil, nil, err
	}

	return readerStdout, readerStderr, nil
}

func getSSHKey(nodeDir string, obj *v3.Node) (string, error) {
	if err := waitUntilSSHKey(nodeDir, obj); err != nil {
		return "", err
	}

	return getSSHPrivateKey(nodeDir, obj)
}

func (m *Lifecycle) reportStatus(stdoutReader io.Reader, stderrReader io.Reader, node *v3.Node) (*v3.Node, error) {
	scanner := bufio.NewScanner(stdoutReader)
	for scanner.Scan() {
		msg := scanner.Text()
		logrus.Infof("stdout: %s", msg)
		_, err := filterDockerMessage(msg, node)
		if err != nil {
			return node, err
		}
		m.logger.Info(node, msg)
		v3.NodeConditionProvisioned.Message(node, msg)
		// ignore update errors
		if newObj, err := m.nodeClient.Update(node); err == nil {
			node = newObj
		} else {
			node, _ = m.nodeClient.Get(node.Name, metav1.GetOptions{})
		}
	}
	scanner = bufio.NewScanner(stderrReader)
	for scanner.Scan() {
		msg := scanner.Text()
		return node, errors.New(msg)
	}
	return node, nil
}

func filterDockerMessage(msg string, node *v3.Node) (string, error) {
	if strings.Contains(msg, errorCreatingNode) {
		return "", errors.New(msg)
	}
	if strings.Contains(msg, node.Spec.RequestedHostname) {
		return "", nil
	}
	return msg, nil
}

func nodeExists(nodeDir string, name string) (bool, error) {
	command := buildCommand(nodeDir, []string{"ls", "-q"})
	r, err := command.StdoutPipe()
	if err != nil {
		return false, err
	}

	err = command.Start()
	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		foundName := scanner.Text()
		if foundName == name {
			return true, nil
		}
	}
	if err = scanner.Err(); err != nil {
		return false, err
	}

	err = command.Wait()
	if err != nil {
		return false, err
	}

	return false, nil
}

func deleteNode(nodeDir string, node *v3.Node) error {
	command := buildCommand(nodeDir, []string{"rm", "-f", node.Spec.RequestedHostname})
	err := command.Start()
	if err != nil {
		return err
	}

	err = command.Wait()
	if err != nil {
		return err
	}

	return nil
}

func getSSHPrivateKey(nodeDir string, node *v3.Node) (string, error) {
	keyPath := filepath.Join(nodeDir, "machines", node.Spec.RequestedHostname, "id_rsa")
	data, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return "", nil
	}
	return string(data), nil
}

func waitUntilSSHKey(nodeDir string, node *v3.Node) error {
	keyPath := filepath.Join(nodeDir, "machines", node.Spec.RequestedHostname, "id_rsa")
	startTime := time.Now()
	increments := 1
	for {
		if time.Now().After(startTime.Add(15 * time.Second)) {
			return errors.New("Timeout waiting for ssh key")
		}
		if _, err := os.Stat(keyPath); err != nil {
			logrus.Debugf("keyPath not found. The node is probably still provisioning. Sleep %v second", increments)
			time.Sleep(time.Duration(increments) * time.Second)
			increments = increments * 2
			continue
		}
		return nil
	}
}
