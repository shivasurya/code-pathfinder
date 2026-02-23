package docker

import (
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// convertFROM extracts FROM instruction details.
// FROM [--platform=<platform>] <image>[:<tag>][@<digest>] [AS <name>].
func convertFROM(node *sitter.Node, source []byte, dn *DockerfileNode) {
	// Extract params (--platform).
	dn.Flags = extractParams(node, source)

	// Find image_spec and alias children.
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "image_spec":
			parseImageSpec(child, source, dn)
		case "image_alias":
			dn.StageAlias = getNodeText(child, source)
		}
	}

	// Default to "latest" if no tag specified.
	if dn.ImageTag == "" && dn.ImageDigest == "" {
		dn.ImageTag = "latest"
	}
}

func parseImageSpec(node *sitter.Node, source []byte, dn *DockerfileNode) {
	text := getNodeText(node, source)

	// Handle digest format: image@sha256:xxx.
	if before, after, ok := strings.Cut(text, "@"); ok {
		dn.BaseImage = before
		dn.ImageDigest = after
		return
	}

	// Handle tag format: image:tag.
	if idx := strings.LastIndex(text, ":"); idx != -1 {
		dn.BaseImage = text[:idx]
		dn.ImageTag = text[idx+1:]
		return
	}

	// No tag or digest.
	dn.BaseImage = text
}

// convertUSER extracts USER instruction details.
// USER <user>[:<group>].
func convertUSER(node *sitter.Node, source []byte, dn *DockerfileNode) {
	// Get full text and parse manually since tree-sitter structure varies.
	raw := getNodeText(node, source)
	raw = strings.TrimPrefix(raw, "USER ")
	raw = strings.TrimSpace(raw)

	if strings.Contains(raw, ":") {
		parts := strings.SplitN(raw, ":", 2)
		dn.UserName = parts[0]
		dn.GroupName = parts[1]
	} else {
		dn.UserName = raw
	}
}

// convertCOPY extracts COPY instruction details.
func convertCOPY(node *sitter.Node, source []byte, dn *DockerfileNode) {
	convertCopyAdd(node, source, dn)
}

// convertADD extracts ADD instruction details.
func convertADD(node *sitter.Node, source []byte, dn *DockerfileNode) {
	convertCopyAdd(node, source, dn)
}

// convertCopyAdd handles both COPY and ADD instructions.
// COPY/ADD [--from=<stage>] [--chown=<user>:<group>] <src>... <dest>.
func convertCopyAdd(node *sitter.Node, source []byte, dn *DockerfileNode) {
	// Extract flags.
	dn.Flags = extractParams(node, source)
	if from, ok := dn.Flags["from"]; ok {
		dn.CopyFrom = from
	}
	if chown, ok := dn.Flags["chown"]; ok {
		dn.Chown = chown
	}

	// Extract paths.
	paths := extractPaths(node, source)
	if len(paths) > 0 {
		dn.DestPath = paths[len(paths)-1]
		if len(paths) > 1 {
			dn.SourcePaths = paths[:len(paths)-1]
		}
	}
}

// convertRUN extracts RUN instruction details.
// RUN <command> or RUN ["executable", "param1", "param2"].
func convertRUN(node *sitter.Node, source []byte, dn *DockerfileNode) {
	dn.Flags = extractParams(node, source)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "shell_command":
			dn.CommandForm = "shell"
			dn.Arguments = []string{getNodeText(child, source)}
		case "json_string_array":
			dn.CommandForm = "exec"
			dn.CommandArray = extractJSONArray(child, source)
		}
	}
}

// convertCMD extracts CMD instruction details.
func convertCMD(node *sitter.Node, source []byte, dn *DockerfileNode) {
	convertCmdEntrypoint(node, source, dn)
}

// convertENTRYPOINT extracts ENTRYPOINT instruction details.
func convertENTRYPOINT(node *sitter.Node, source []byte, dn *DockerfileNode) {
	convertCmdEntrypoint(node, source, dn)
}

// convertCmdEntrypoint handles both CMD and ENTRYPOINT instructions.
func convertCmdEntrypoint(node *sitter.Node, source []byte, dn *DockerfileNode) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "shell_command":
			dn.CommandForm = "shell"
			dn.Arguments = []string{getNodeText(child, source)}
		case "json_string_array":
			dn.CommandForm = "exec"
			dn.CommandArray = extractJSONArray(child, source)
		}
	}
}

// convertENV extracts ENV instruction details.
// ENV <key>=<value> ... or ENV <key> <value>.
func convertENV(node *sitter.Node, source []byte, dn *DockerfileNode) {
	// Parse from raw text since tree-sitter structure can vary.
	raw := getNodeText(node, source)
	raw = strings.TrimPrefix(raw, "ENV ")
	raw = strings.TrimSpace(raw)

	// Split on whitespace to handle multiple env vars.
	parts := strings.FieldsSeq(raw)
	for part := range parts {
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key := strings.Trim(kv[0], "\"")
			value := strings.Trim(kv[1], "\"")
			dn.EnvVars[key] = value
		}
	}
}

// convertARG extracts ARG instruction details.
// ARG <name>[=<default>].
func convertARG(node *sitter.Node, source []byte, dn *DockerfileNode) {
	// Parse from raw text.
	raw := getNodeText(node, source)
	raw = strings.TrimPrefix(raw, "ARG ")
	raw = strings.TrimSpace(raw)

	if strings.Contains(raw, "=") {
		parts := strings.SplitN(raw, "=", 2)
		dn.ArgName = parts[0]
		dn.Arguments = []string{parts[1]}
	} else {
		dn.ArgName = raw
	}
}

// convertEXPOSE extracts EXPOSE instruction details.
// EXPOSE <port>[/<protocol>] ....
func convertEXPOSE(node *sitter.Node, source []byte, dn *DockerfileNode) {
	dn.Ports = make([]int, 0)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "expose_port" {
			text := getNodeText(child, source)
			if strings.Contains(text, "/") {
				parts := strings.SplitN(text, "/", 2)
				if port, err := strconv.Atoi(parts[0]); err == nil {
					dn.Ports = append(dn.Ports, port)
				}
				dn.Protocol = parts[1]
			} else {
				if port, err := strconv.Atoi(text); err == nil {
					dn.Ports = append(dn.Ports, port)
				}
			}
		}
	}
}

// convertWORKDIR extracts WORKDIR instruction details.
// WORKDIR /path/to/workdir.
func convertWORKDIR(node *sitter.Node, source []byte, dn *DockerfileNode) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "path" {
			dn.WorkDir = getNodeText(child, source)
			dn.IsAbsolutePath = strings.HasPrefix(dn.WorkDir, "/")
			break
		}
	}
}

// convertVOLUME extracts VOLUME instruction details.
// VOLUME ["/data"] or VOLUME /data /logs.
func convertVOLUME(node *sitter.Node, source []byte, dn *DockerfileNode) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "json_string_array":
			dn.Volumes = extractJSONArray(child, source)
		case "path":
			dn.Volumes = append(dn.Volumes, getNodeText(child, source))
		}
	}
}

// convertSHELL extracts SHELL instruction details.
// SHELL ["executable", "param"].
func convertSHELL(node *sitter.Node, source []byte, dn *DockerfileNode) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "json_string_array" {
			dn.Shell = extractJSONArray(child, source)
			break
		}
	}
}

// convertHEALTHCHECK extracts HEALTHCHECK instruction details.
// HEALTHCHECK [OPTIONS] CMD command or HEALTHCHECK NONE.
func convertHEALTHCHECK(node *sitter.Node, source []byte, dn *DockerfileNode) {
	dn.Flags = extractParams(node, source)

	// Map flags to structured fields.
	if interval, ok := dn.Flags["interval"]; ok {
		dn.HealthcheckInterval = interval
	}
	if timeout, ok := dn.Flags["timeout"]; ok {
		dn.HealthcheckTimeout = timeout
	}
	if startPeriod, ok := dn.Flags["start-period"]; ok {
		dn.HealthcheckStartPeriod = startPeriod
	}
	if retries, ok := dn.Flags["retries"]; ok {
		if n, err := strconv.Atoi(retries); err == nil {
			dn.HealthcheckRetries = n
		}
	}

	// Check for NONE.
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		text := getNodeText(child, source)
		if text == "NONE" {
			dn.HealthcheckType = "NONE"
		}
		if child.Type() == "cmd_instruction" {
			dn.HealthcheckType = "CMD"
			dn.HealthcheckCmd = getNodeText(child, source)
		}
	}
}

// convertLABEL extracts LABEL instruction details.
// LABEL <key>=<value> ....
func convertLABEL(node *sitter.Node, source []byte, dn *DockerfileNode) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "label_pair" {
			key := ""
			value := ""
			for j := 0; j < int(child.ChildCount()); j++ {
				pair := child.Child(j)
				text := getNodeText(pair, source)
				if key == "" {
					key = strings.Trim(text, "\"'")
				} else {
					value = strings.Trim(text, "\"'")
				}
			}
			if key != "" {
				dn.Labels[key] = value
			}
		}
	}
}

// convertONBUILD extracts ONBUILD instruction details.
// ONBUILD <INSTRUCTION>.
func convertONBUILD(node *sitter.Node, source []byte, dn *DockerfileNode) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if isInstructionNode(child) {
			dn.OnBuildInstruction = getNodeText(child, source)
			break
		}
	}
}

// convertSTOPSIGNAL extracts STOPSIGNAL instruction details.
// STOPSIGNAL signal.
func convertSTOPSIGNAL(node *sitter.Node, source []byte, dn *DockerfileNode) {
	raw := getNodeText(node, source)
	dn.StopSignal = strings.TrimPrefix(raw, "STOPSIGNAL ")
	dn.StopSignal = strings.TrimSpace(dn.StopSignal)
}

// convertMAINTAINER extracts MAINTAINER instruction details.
// MAINTAINER <name>.
func convertMAINTAINER(node *sitter.Node, source []byte, dn *DockerfileNode) {
	raw := getNodeText(node, source)
	maintainer := strings.TrimPrefix(raw, "MAINTAINER ")
	maintainer = strings.TrimSpace(maintainer)
	dn.Arguments = []string{maintainer}
}

// Helper functions.

// extractParams extracts all --flag=value params from an instruction node.
func extractParams(node *sitter.Node, source []byte) map[string]string {
	params := make(map[string]string)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "param" {
			text := getNodeText(child, source)
			// Parse --name=value.
			text = strings.TrimPrefix(text, "--")
			if before, after, ok := strings.Cut(text, "="); ok {
				params[before] = after
			}
		}
	}

	return params
}

// extractPaths extracts all path nodes from an instruction.
func extractPaths(node *sitter.Node, source []byte) []string {
	paths := make([]string, 0)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "path" {
			paths = append(paths, getNodeText(child, source))
		}
	}

	return paths
}

// extractJSONArray extracts strings from a JSON array node.
func extractJSONArray(node *sitter.Node, source []byte) []string {
	items := make([]string, 0)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "json_string" || child.Type() == "double_quoted_string" {
			text := getNodeText(child, source)
			text = strings.Trim(text, "\"")
			items = append(items, text)
		}
	}

	return items
}

// getNodeText safely extracts text from a tree-sitter node.
func getNodeText(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	return node.Content(source)
}
