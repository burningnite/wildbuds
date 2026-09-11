package commands_test

import (
	"encoding/json"
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestCommandType_Validation(t *testing.T) {
	valid := []commands.CommandType{
		commands.CommandSelectUnit,
		commands.CommandMoveUnit,
		commands.CommandAttack,
		commands.CommandEndActivation,
	}

	for _, ct := range valid {
		if !ct.IsValid() {
			t.Errorf("expected %s to be valid", ct)
		}
		if ct.String() != string(ct) {
			t.Errorf("expected String() == %s, got %s", ct, ct.String())
		}
	}

	invalid := []commands.CommandType{
		"",
		"Select",
		"MOVE",
		"random_string",
	}

	for _, ct := range invalid {
		if ct.IsValid() {
			t.Errorf("expected %q to be invalid", ct)
		}
	}
}

func TestCommandType_JSONSerialization(t *testing.T) {
	ct := commands.CommandSelectUnit
	data, err := json.Marshal(ct)
	if err != nil {
		t.Fatalf("failed to marshal CommandType: %v", err)
	}
	if string(data) != `"SelectUnit"` {
		t.Errorf("unexpected marshaled CommandType: %s", string(data))
	}

	var decoded commands.CommandType
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CommandType: %v", err)
	}
	if decoded != ct {
		t.Errorf("expected %s, got %s", ct, decoded)
	}

	// Invalid type unmarshal
	invalidData := []byte(`"BogusCommand"`)
	if err := json.Unmarshal(invalidData, &decoded); err == nil {
		t.Errorf("expected error unmarshaling bogus command type, got nil")
	}

	// Invalid type marshal
	invalidCT := commands.CommandType("Bogus")
	if _, err := json.Marshal(invalidCT); err == nil {
		t.Errorf("expected error marshaling invalid command type, got nil")
	}
}

func TestGameCommand_ConstructorsAndValidation(t *testing.T) {
	// SelectUnit valid
	cmdSel := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	if err := cmdSel.Validate(); err != nil {
		t.Fatalf("expected valid SelectUnit, got %v", err)
	}
	if cmdSel.Type != commands.CommandSelectUnit {
		t.Errorf("expected CommandSelectUnit, got %s", cmdSel.Type)
	}

	// SelectUnit out of bounds
	cmdSelBad := commands.NewSelectUnitCommand(domain.GridPosition{X: 10, Y: 0})
	if err := cmdSelBad.Validate(); err != commands.ErrTargetOutOfBounds {
		t.Fatalf("expected ErrTargetOutOfBounds, got %v", err)
	}

	// MoveUnit valid
	cmdMove := commands.NewMoveUnitCommand(domain.GridPosition{X: 1, Y: 1})
	if err := cmdMove.Validate(); err != nil {
		t.Fatalf("expected valid MoveUnit, got %v", err)
	}

	// MoveUnit out of bounds
	cmdMoveBad := commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -6})
	if err := cmdMoveBad.Validate(); err != commands.ErrDestOutOfBounds {
		t.Fatalf("expected ErrDestOutOfBounds, got %v", err)
	}

	// Attack valid
	cmdAtk := commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 3})
	if err := cmdAtk.Validate(); err != nil {
		t.Fatalf("expected valid Attack, got %v", err)
	}

	// Attack out of bounds
	cmdAtkBad := commands.NewAttackCommand(domain.GridPosition{X: -6, Y: 0})
	if err := cmdAtkBad.Validate(); err != commands.ErrTargetOutOfBounds {
		t.Fatalf("expected ErrTargetOutOfBounds, got %v", err)
	}

	// EndActivation valid
	cmdEnd := commands.NewEndActivationCommand()
	if err := cmdEnd.Validate(); err != nil {
		t.Fatalf("expected valid EndActivation, got %v", err)
	}

	// Empty / invalid command type
	emptyCmd := commands.GameCommand{}
	if err := emptyCmd.Validate(); err == nil {
		t.Fatalf("expected error for empty command, got nil")
	}
}

func TestGameCommand_JSON_RoundTrip_CanonicalFlat(t *testing.T) {
	tests := []struct {
		name string
		cmd  commands.GameCommand
	}{
		{
			name: "SelectUnit",
			cmd:  commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}),
		},
		{
			name: "MoveUnit",
			cmd:  commands.NewMoveUnitCommand(domain.GridPosition{X: 2, Y: 1}),
		},
		{
			name: "Attack",
			cmd:  commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 3}),
		},
		{
			name: "EndActivation",
			cmd:  commands.NewEndActivationCommand(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.cmd)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var decoded commands.GameCommand
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if !decoded.Equals(tc.cmd) {
				t.Errorf("mismatch:\nexpected: %#v\ngot:      %#v", tc.cmd, decoded)
			}
			if err := decoded.Validate(); err != nil {
				t.Errorf("decoded command invalid: %v", err)
			}
		})
	}
}

func TestGameCommand_JSON_RustSerdeCompat(t *testing.T) {
	// 1. Rust externally tagged SelectUnit
	rustSel := []byte(`{"SelectUnit":{"target":{"x":0,"y":-3}}}`)
	var cmdSel commands.GameCommand
	if err := json.Unmarshal(rustSel, &cmdSel); err != nil {
		t.Fatalf("failed to unmarshal Rust SelectUnit: %v", err)
	}
	expectedSel := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	if !cmdSel.Equals(expectedSel) {
		t.Errorf("expected %v, got %v", expectedSel, cmdSel)
	}

	// 2. Rust externally tagged MoveUnit
	rustMove := []byte(`{"MoveUnit":{"destination":{"x":-2,"y":4}}}`)
	var cmdMove commands.GameCommand
	if err := json.Unmarshal(rustMove, &cmdMove); err != nil {
		t.Fatalf("failed to unmarshal Rust MoveUnit: %v", err)
	}
	expectedMove := commands.NewMoveUnitCommand(domain.GridPosition{X: -2, Y: 4})
	if !cmdMove.Equals(expectedMove) {
		t.Errorf("expected %v, got %v", expectedMove, cmdMove)
	}

	// 3. Rust externally tagged Attack
	rustAtk := []byte(`{"Attack":{"target":{"x":0,"y":3}}}`)
	var cmdAtk commands.GameCommand
	if err := json.Unmarshal(rustAtk, &cmdAtk); err != nil {
		t.Fatalf("failed to unmarshal Rust Attack: %v", err)
	}
	expectedAtk := commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 3})
	if !cmdAtk.Equals(expectedAtk) {
		t.Errorf("expected %v, got %v", expectedAtk, cmdAtk)
	}

	// 4. Rust string unit variant: "EndActivation"
	rustEndString := []byte(`"EndActivation"`)
	var cmdEndStr commands.GameCommand
	if err := json.Unmarshal(rustEndString, &cmdEndStr); err != nil {
		t.Fatalf("failed to unmarshal Rust string EndActivation: %v", err)
	}
	expectedEnd := commands.NewEndActivationCommand()
	if !cmdEndStr.Equals(expectedEnd) {
		t.Errorf("expected %v, got %v", expectedEnd, cmdEndStr)
	}

	// 5. Rust object variant: {"EndActivation":{}}
	rustEndObj := []byte(`{"EndActivation":{}}`)
	var cmdEndObj commands.GameCommand
	if err := json.Unmarshal(rustEndObj, &cmdEndObj); err != nil {
		t.Fatalf("failed to unmarshal Rust object EndActivation: %v", err)
	}
	if !cmdEndObj.Equals(expectedEnd) {
		t.Errorf("expected %v, got %v", expectedEnd, cmdEndObj)
	}
}

func TestGameCommand_JSON_Malformed(t *testing.T) {
	var cmd commands.GameCommand

	// Empty data
	if err := json.Unmarshal([]byte(``), &cmd); err == nil {
		t.Errorf("expected error on empty data")
	}

	// Whitespace only
	if err := json.Unmarshal([]byte(`   `), &cmd); err == nil {
		t.Errorf("expected error on whitespace data")
	}

	// Invalid JSON syntax
	if err := json.Unmarshal([]byte(`{not valid json}`), &cmd); err == nil {
		t.Errorf("expected error on invalid JSON syntax")
	}

	// Invalid unit string variant
	if err := json.Unmarshal([]byte(`"InvalidUnit"`), &cmd); err == nil {
		t.Errorf("expected error on invalid unit string")
	}

	// Multiple keys in externally tagged format
	if err := json.Unmarshal([]byte(`{"SelectUnit":{},"MoveUnit":{}}`), &cmd); err == nil {
		t.Errorf("expected error on multiple keys in externally tagged format")
	}

	// Invalid command type in flat format
	if err := json.Unmarshal([]byte(`{"type":"NonexistentCommand"}`), &cmd); err == nil {
		t.Errorf("expected error on nonexistent command type in flat format")
	}

	// Unknown key in externally tagged format
	if err := json.Unmarshal([]byte(`{"NonexistentCommand":{}}`), &cmd); err == nil {
		t.Errorf("expected error on unknown key in externally tagged format")
	}

	// Marshaling invalid command
	invalidCmd := commands.GameCommand{Type: commands.CommandType("Invalid")}
	if _, err := json.Marshal(invalidCmd); err == nil {
		t.Errorf("expected error marshaling invalid command")
	}
}

func TestGameCommand_Equals_And_String(t *testing.T) {
	c1 := commands.NewSelectUnitCommand(domain.GridPosition{X: 1, Y: 2})
	c2 := commands.NewSelectUnitCommand(domain.GridPosition{X: 1, Y: 2})
	c3 := commands.NewSelectUnitCommand(domain.GridPosition{X: 1, Y: 3})
	c4 := commands.NewMoveUnitCommand(domain.GridPosition{X: 1, Y: 2})
	c5 := commands.NewEndActivationCommand()
	c6 := commands.NewEndActivationCommand()
	c7 := commands.NewAttackCommand(domain.GridPosition{X: 1, Y: 2})

	if !c1.Equals(c2) {
		t.Errorf("expected c1 == c2")
	}
	if c1.Equals(c3) {
		t.Errorf("expected c1 != c3")
	}
	if c1.Equals(c4) {
		t.Errorf("expected c1 != c4 (type difference)")
	}
	if !c5.Equals(c6) {
		t.Errorf("expected c5 == c6 (EndActivation)")
	}
	if c1.Equals(c7) {
		t.Errorf("expected c1 != c7")
	}

	// String representations
	if c1.String() != "SelectUnit(target=(1, 2))" {
		t.Errorf("unexpected string: %s", c1.String())
	}
	if c4.String() != "MoveUnit(destination=(1, 2))" {
		t.Errorf("unexpected string: %s", c4.String())
	}
	if c7.String() != "Attack(target=(1, 2))" {
		t.Errorf("unexpected string: %s", c7.String())
	}
	if c5.String() != "EndActivation" {
		t.Errorf("unexpected string: %s", c5.String())
	}
	unknownCmd := commands.GameCommand{Type: "Custom"}
	if unknownCmd.String() != "UnknownCommand(Custom)" {
		t.Errorf("unexpected string: %s", unknownCmd.String())
	}
}

func TestNetworkCommand_ValidationAndJSON(t *testing.T) {
	cmd := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	netCmd := commands.NewNetworkCommand(domain.PlayerOne, cmd)

	if err := netCmd.Validate(); err != nil {
		t.Fatalf("expected valid NetworkCommand, got %v", err)
	}

	// Invalid sender
	badNetCmd := commands.NewNetworkCommand(domain.PlayerNone, cmd)
	if err := badNetCmd.Validate(); err != commands.ErrInvalidSender {
		t.Fatalf("expected ErrInvalidSender, got %v", err)
	}

	// Invalid inner command
	badInnerCmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 10, Y: 0}))
	if err := badInnerCmd.Validate(); err != commands.ErrTargetOutOfBounds {
		t.Fatalf("expected ErrTargetOutOfBounds, got %v", err)
	}

	// Equals
	netCmd2 := commands.NewNetworkCommand(domain.PlayerOne, cmd)
	netCmdP2 := commands.NewNetworkCommand(domain.PlayerTwo, cmd)
	if !netCmd.Equals(netCmd2) {
		t.Errorf("expected netCmd == netCmd2")
	}
	if netCmd.Equals(netCmdP2) {
		t.Errorf("expected netCmd != netCmdP2")
	}

	// String
	str := netCmd.String()
	if str != "NetworkCommand(sender=PlayerOne, cmd=SelectUnit(target=(0, -3)))" {
		t.Errorf("unexpected string: %s", str)
	}

	// JSON Round trip
	data, err := json.Marshal(netCmd)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded commands.NetworkCommand
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !decoded.Equals(netCmd) {
		t.Errorf("mismatch:\nexpected: %s\ngot:      %s", netCmd, decoded)
	}
}
