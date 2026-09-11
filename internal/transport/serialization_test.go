package transport_test

import (
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/transport"
)

func TestSerialization_GameCommandSymmetry(t *testing.T) {
	testCases := []struct {
		name string
		cmd  commands.GameCommand
	}{
		{
			name: "SelectUnit",
			cmd:  commands.NewSelectUnitCommand(domain.GridPosition{X: 1, Y: -2}),
		},
		{
			name: "MoveUnit",
			cmd:  commands.NewMoveUnitCommand(domain.GridPosition{X: 3, Y: 4}),
		},
		{
			name: "Attack",
			cmd:  commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 0}),
		},
		{
			name: "EndActivation",
			cmd:  commands.NewEndActivationCommand(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := transport.SerializeGameCommand(tc.cmd)
			if err != nil {
				t.Fatalf("SerializeGameCommand failed: %v", err)
			}

			deserialized, err := transport.DeserializeGameCommand(data)
			if err != nil {
				t.Fatalf("DeserializeGameCommand failed: %v", err)
			}

			if !tc.cmd.Equals(deserialized) {
				t.Errorf("roundtrip symmetry failed: expected %s, got %s", tc.cmd, deserialized)
			}
		})
	}
}

func TestSerialization_NetworkCommandSymmetry(t *testing.T) {
	gameCmd := commands.NewMoveUnitCommand(domain.GridPosition{X: -3, Y: 5})
	netCmd := commands.NewNetworkCommand(domain.PlayerOne, gameCmd)

	data, err := transport.SerializeNetworkCommand(netCmd)
	if err != nil {
		t.Fatalf("SerializeNetworkCommand failed: %v", err)
	}

	deserialized, err := transport.DeserializeNetworkCommand(data)
	if err != nil {
		t.Fatalf("DeserializeNetworkCommand failed: %v", err)
	}

	if !netCmd.Equals(deserialized) {
		t.Errorf("network command roundtrip symmetry failed: expected %s, got %s", netCmd, deserialized)
	}
}

func TestSerialization_Errors(t *testing.T) {
	badData := []byte("{invalid-json")
	_, err := transport.DeserializeGameCommand(badData)
	if err == nil {
		t.Errorf("expected error when deserializing invalid JSON into GameCommand, got nil")
	}

	_, err = transport.DeserializeNetworkCommand(badData)
	if err == nil {
		t.Errorf("expected error when deserializing invalid JSON into NetworkCommand, got nil")
	}
}
