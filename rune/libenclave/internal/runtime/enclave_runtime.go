package runtime // import "github.com/opencontainers/runc/libenclave/internal/runtime"

import (
	"github.com/opencontainers/runc/libenclave/configs"
	core "github.com/opencontainers/runc/libenclave/internal/runtime/core"
	pal "github.com/opencontainers/runc/libenclave/internal/runtime/pal"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

type EnclaveRuntime interface {
	Name() string
	Load(path string) error
	Init(args string, logLevel string) error
	Attest() error
	Exec(cmd []string, envp []string, stdio [3]*os.File) (int32, error)
	Kill(sig int, pid int) error
	Destroy() error
}

type EnclaveRuntimeWrapper struct {
	runtime EnclaveRuntime
}

func StartInitialization(config *configs.InitEnclaveConfig, logLevel string) (*EnclaveRuntimeWrapper, error) {
	logrus.Debugf("enclave init config retrieved: %+v", config)

	var (
		runtime EnclaveRuntime
		err     error
	)
	runtime, err = core.StartInitialization(config)
	if err != nil {
		runtime, err = pal.StartInitialization(config)
		if err != nil {
			return nil, err
		}
	}

	logrus.Infof("Loading enclave runtime %s", config.Path)
	err = runtime.Load(config.Path)
	if err != nil {
		return nil, err
	}
	name := runtime.Name()

	logrus.Infof("Initializing enclave runtime %s", name)
	err = runtime.Init(config.Args, logLevel)
	if err != nil {
		return nil, err
	}

	rt := &EnclaveRuntimeWrapper{
		runtime: runtime,
	}
	return rt, nil
}

func (rt *EnclaveRuntimeWrapper) LaunchAttestation() error {
	logrus.Debugf("attesting enclave runtime %s", rt.runtime.Name())

	return rt.runtime.Attest()
}

func (rt *EnclaveRuntimeWrapper) ExecutePayload(cmd []string, envp []string, stdio [3]*os.File) (int32, error) {
	logrus.Debugf("enclave runtime %s executing payload with commandline %s", rt.runtime.Name(), cmd)

	// The executable may not exist in container at all according
	// to the design of enclave runtime, such as Occlum, which uses
	// an invisible filesystem to the container. In this case, the
	// lookup will fail.
	if fullPath, err := exec.LookPath(cmd[0]); err == nil {
		cmd[0] = fullPath
	}
	return rt.runtime.Exec(cmd, envp, stdio)
}

func (rt *EnclaveRuntimeWrapper) KillPayload(sig int, pid int) error {
	if pid != -1 {
		logrus.Debugf("enclave runtime %s killing payload %d with signal %d", rt.runtime.Name(), pid, sig)
	} else {
		logrus.Debugf("enclave runtime %s killing all payloads with signal %d", rt.runtime.Name(), sig)
	}

	return rt.runtime.Kill(sig, pid)
}

func (rt *EnclaveRuntimeWrapper) DestroyInstance() error {
	logrus.Debugf("Destroying enclave runtime %s", rt.runtime.Name())

	return rt.runtime.Destroy()
}
