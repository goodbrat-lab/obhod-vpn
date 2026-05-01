package network

import (
	"strings"
	"testing"
)

func TestNetworkManager_ApplyRules(t *testing.T) {
	nm := NewNetworkManager(255, 11080)

	var executedCommands []string
	nm.ExecCmd = func(name string, args ...string) error {
		cmdStr := name + " " + strings.Join(args, " ")
		executedCommands = append(executedCommands, cmdStr)
		return nil
	}

	err := nm.ApplyRules()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(executedCommands) != 3 {
		t.Fatalf("expected 3 commands to be executed, got %d", len(executedCommands))
	}

	if executedCommands[0] != "ip route add local default dev lo table 105" {
		t.Errorf("unexpected first command: %s", executedCommands[0])
	}
	if executedCommands[1] != "ip rule add fwmark 255/255 table 105" {
		t.Errorf("unexpected second command: %s", executedCommands[1])
	}
	if !strings.HasPrefix(executedCommands[2], "nft -f") {
		t.Errorf("unexpected third command: %s", executedCommands[2])
	}
}

func TestNetworkManager_Flush(t *testing.T) {
	nm := NewNetworkManager(255, 11080)

	var executedCommands []string
	nm.ExecCmd = func(name string, args ...string) error {
		cmdStr := name + " " + strings.Join(args, " ")
		executedCommands = append(executedCommands, cmdStr)
		return nil
	}

	nm.Flush()

	if len(executedCommands) != 3 {
		t.Fatalf("expected 3 commands to be executed, got %d", len(executedCommands))
	}

	if executedCommands[0] != "nft delete table inet obhod" {
		t.Errorf("unexpected first command: %s", executedCommands[0])
	}
	if executedCommands[1] != "ip rule del fwmark 255/255 table 105" {
		t.Errorf("unexpected second command: %s", executedCommands[1])
	}
	if executedCommands[2] != "ip route del local default dev lo table 105" {
		t.Errorf("unexpected third command: %s", executedCommands[2])
	}
}
