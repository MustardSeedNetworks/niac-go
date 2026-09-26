package daemon

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// PinAttachmentClient fixes one client MAC to one port of the session's
// attachment pool and restarts the session on it. The pin is written into the
// scenario file the session runs, the same `attachments[].pins` the editor
// authors, so the move survives the restart and the next start alike.
//
// The restart is an ordinary start of the same session, so the amended scenario
// passes every check a start makes: a pin outside the pool, or one that collides
// with another pin, is refused by the compile before the running session is
// replaced. Any refusal puts the file back as it was.
func (d *Daemon) PinAttachmentClient(sessionID string, pin api.AttachmentPin) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	active := d.sessions.get(sessionID)
	if active == nil {
		return api.ErrSimulationSessionNotFound
	}
	req := active.Request
	if req.Attachment == "" {
		return api.ErrAttachmentPoolRequired
	}
	generation, err := newRuntimeGeneration()
	if err != nil {
		return err
	}
	original, err := os.ReadFile(active.ConfigPath)
	if err != nil {
		return fmt.Errorf("read scenario: %w", err)
	}
	amended, err := setAttachmentPin(original, req.Attachment, config.AttachmentPin(pin))
	if err != nil {
		return err
	}
	directory, name := filepath.Split(active.ConfigPath)
	if err = writeStateFile(directory, name, amended); err != nil {
		return fmt.Errorf("write scenario: %w", err)
	}
	if err = d.startGenerationLocked(req, generation, false); err != nil {
		return d.restoreScenario(directory, name, original, err)
	}
	return nil
}

func (d *Daemon) restoreScenario(directory, name string, original []byte, cause error) error {
	if restoreErr := writeStateFile(directory, name, original); restoreErr != nil {
		return fmt.Errorf("%w (restoring the scenario also failed: %w)", cause, restoreErr)
	}
	return cause
}

// setAttachmentPin edits the scenario as a YAML document rather than through
// the typed config, so every field the pin does not concern is written back as
// it was read. Any earlier pin of the same MAC is replaced: a client sits on
// one port.
func setAttachmentPin(content []byte, attachment string, pin config.AttachmentPin) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, api.ErrAttachmentPoolRequired
	}
	target := findAttachment(mappingValue(document.Content[0], "attachments"), attachment)
	if target == nil || mappingValue(target, "at") == nil {
		return nil, api.ErrAttachmentPoolRequired
	}
	pins := mappingValue(target, "pins")
	if pins == nil {
		pins = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		target.Content = append(target.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "pins"}, pins)
	}
	pins.Content = slices.DeleteFunc(pins.Content, func(existing *yaml.Node) bool {
		return sameMAC(scalarValue(existing, "mac"), pin.MAC)
	})
	pins.Content = append(pins.Content, pinNode(pin))

	out, err := yaml.Marshal(&document)
	if err != nil {
		return nil, fmt.Errorf("serialize scenario: %w", err)
	}
	return out, nil
}

func findAttachment(attachments *yaml.Node, name string) *yaml.Node {
	if attachments == nil || attachments.Kind != yaml.SequenceNode {
		return nil
	}
	for _, item := range attachments.Content {
		if item.Kind == yaml.MappingNode && scalarValue(item, "name") == name {
			return item
		}
	}
	return nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func scalarValue(mapping *yaml.Node, key string) string {
	if value := mappingValue(mapping, key); value != nil && value.Kind == yaml.ScalarNode {
		return value.Value
	}
	return ""
}

func sameMAC(a, b string) bool {
	parsedA, errA := net.ParseMAC(a)
	parsedB, errB := net.ParseMAC(b)
	return errA == nil && errB == nil && bytes.Equal(parsedA, parsedB)
}

func pinNode(pin config.AttachmentPin) *yaml.Node {
	scalar := func(value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	}
	// Quoted as the editor writes it: a YAML 1.1 reader takes a bare
	// colon-separated MAC for a base-60 number.
	mac := scalar(pin.MAC)
	mac.Style = yaml.DoubleQuotedStyle
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		scalar("mac"), mac,
		scalar("device"), scalar(pin.Device),
		scalar("interface"), scalar(pin.Interface),
	}}
}
