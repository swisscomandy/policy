package iptables_manager

import (
	"bytes"
	"io/ioutil"
	"net"
        "strings"
	"os/exec"

	"fmt"
	"code.cloudfoundry.org/garden-linux/sysconfig"
	"github.com/cloudfoundry/gunk/command_runner"
	"code.cloudfoundry.org/lager"
)

type natChain struct {
	cfg    *sysconfig.IPTablesNATConfig
	runner command_runner.CommandRunner
	logger lager.Logger
}

func NewNATChain(cfg *sysconfig.IPTablesNATConfig, runner command_runner.CommandRunner, logger lager.Logger) *natChain {
	return &natChain{
		cfg:    cfg,
		runner: runner,
		logger: logger,
	}
}

func (mgr *natChain) Setup(containerID, bridgeName string, ip net.IP, network *net.IPNet, externalIp net.IP, portRange string) error {
	instanceChain := mgr.cfg.InstancePrefix + containerID
        commands := []*exec.Cmd{
		// Create nat instance chain
		exec.Command("iptables", "--wait", "--table", "nat", "-N", instanceChain),
		// Bind nat instance chain to nat prerouting chain
		exec.Command("iptables", "--wait", "--table", "nat", "-A", mgr.cfg.PreroutingChain, "--jump", instanceChain),
	}
        condition := "-A w--postrouting -s "+ ip.String() + "/32" + " -p udp -j SNAT --to-source "+ externalIp.String() +":"+ portRange
        if isNewRule(condition) {
	       commands = append(commands, exec.Command("iptables", "--wait", "--table", "nat", "-A", mgr.cfg.PostroutingChain, "--protocol", "tcp", "-s", ip.String(), "-j", "SNAT", "--to", externalIp.String()+":"+portRange),
			exec.Command("iptables", "--wait", "--table", "nat", "-A", mgr.cfg.PostroutingChain, "--protocol", "udp", "-s", ip.String(), "-j", "SNAT", "--to", externalIp.String()+":"+portRange),
			exec.Command("iptables", "--wait", "--table", "nat", "-A", mgr.cfg.PostroutingChain, "--protocol", "icmp", "-s", ip.String(), "-j", "MASQUERADE"))
	}
    
	for _, cmd := range commands {
		if err := mgr.runner.Run(cmd); err != nil {
			buffer := &bytes.Buffer{}
			cmd.Stderr = buffer
			logger := mgr.logger.Session("setup", lager.Data{"cmd": cmd})
			logger.Debug("starting")
			if err := mgr.runner.Run(cmd); err != nil {
				stderr, _ := ioutil.ReadAll(buffer)
				logger.Error("failed", err, lager.Data{"stderr": string(stderr)})
				return fmt.Errorf("iptables_manager: nat: %s", err)
			}
			logger.Debug("ended")
		}
	}

	return nil
}

func isNewRule(condition string) bool {
	out, _ := exec.Command("iptables-save", "-t", "nat").CombinedOutput()
		for _, line := range strings.Split(string(out), "\n") {
        	if strings.EqualFold(line, condition) {
        		return false
                	 }
        	}
	return true
}

func (mgr *natChain) Teardown(containerID string) error {
	instanceChain := mgr.cfg.InstancePrefix + containerID

	commands := []*exec.Cmd{
		// Prune nat prerouting chain
		exec.Command("sh", "-c", fmt.Sprintf(
			`iptables --wait --table nat -S %s 2> /dev/null | grep "\-j %s\b" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 iptables --wait --table nat`,
			mgr.cfg.PreroutingChain, instanceChain,
		)),
		// Flush nat instance chain
		exec.Command("sh", "-c", fmt.Sprintf(`iptables --wait --table nat -F %s 2> /dev/null || true`, instanceChain)),
		// Delete nat instance chain
		exec.Command("sh", "-c", fmt.Sprintf(`iptables --wait --table nat -X %s 2> /dev/null || true`, instanceChain)),
	}

	for _, cmd := range commands {
		buffer := &bytes.Buffer{}
		cmd.Stderr = buffer
		logger := mgr.logger.Session("teardown", lager.Data{"cmd": cmd})
		logger.Debug("starting")
		if err := mgr.runner.Run(cmd); err != nil {
			stderr, _ := ioutil.ReadAll(buffer)
			logger.Error("failed", err, lager.Data{"stderr": string(stderr)})
			return fmt.Errorf("iptables_manager: nat: %s", err)
		}
		logger.Debug("ended")
	}

	return nil
}
