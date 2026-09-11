package transport

import (
	"bytes"
	"encoding/json"
	"fmt"

	"wildbuds/internal/commands"
)

// SerializeGameCommand converts a GameCommand into bytes for network transmission.
func SerializeGameCommand(cmd commands.GameCommand) ([]byte, error) {
	return json.Marshal(cmd)
}

// DeserializeGameCommand converts bytes back into a GameCommand.
func DeserializeGameCommand(data []byte) (commands.GameCommand, error) {
	var cmd commands.GameCommand
	err := json.Unmarshal(data, &cmd)
	return cmd, err
}

// SerializeCommand is an alias for SerializeGameCommand.
func SerializeCommand(cmd commands.GameCommand) ([]byte, error) {
	return SerializeGameCommand(cmd)
}

// DeserializeCommand is an alias for DeserializeGameCommand.
func DeserializeCommand(data []byte) (commands.GameCommand, error) {
	return DeserializeGameCommand(data)
}

// SerializeNetworkCommand converts a NetworkCommand into bytes for network transmission.
func SerializeNetworkCommand(nc commands.NetworkCommand) ([]byte, error) {
	return json.Marshal(nc)
}

// DeserializeNetworkCommand converts bytes back into a NetworkCommand.
// Supports both NetworkCommand JSON and raw GameCommand JSON (defaulting sender if needed).
func DeserializeNetworkCommand(data []byte) (commands.NetworkCommand, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return commands.NetworkCommand{}, fmt.Errorf("empty command data")
	}

	// Try unmarshaling as NetworkCommand first
	var nc commands.NetworkCommand
	if err := json.Unmarshal(trimmed, &nc); err == nil && nc.Sender.IsValid() && nc.Command.Type.IsValid() {
		return nc, nil
	}

	// Fallback: try unmarshaling as GameCommand directly
	var gc commands.GameCommand
	if err := json.Unmarshal(trimmed, &gc); err == nil && gc.Type.IsValid() {
		return commands.NewNetworkCommand(0, gc), nil
	}

	return commands.NetworkCommand{}, fmt.Errorf("failed to deserialize NetworkCommand from: %s", string(trimmed))
}
