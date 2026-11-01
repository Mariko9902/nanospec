package validator

import (
	"regexp"
	"strings"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

const acceptanceIDExpression = `A[1-9]\d*`

var (
	canonicalAcceptanceID   = regexp.MustCompile(`^` + acceptanceIDExpression + `$`)
	acceptanceIDPattern     = regexp.MustCompile(`(?m)^\s*-\s+\[[ xX]\]\s+(` + acceptanceIDExpression + `):\s+\S`)
	acceptanceItemPattern   = regexp.MustCompile(`^\s*-\s+\[[ xX]\]\s+(` + acceptanceIDExpression + `):`)
	acceptanceTokenPattern  = regexp.MustCompile(`(?m)^\s*-\s+\[[ xX]\]\s+([^:\s]+):`)
	acceptanceChecksPattern = regexp.MustCompile(`^\s+-\s+Checks:\s*(.*?)\s*$`)
	checkStartPattern       = regexp.MustCompile(`^\s*-\s+Check:\s*(.*?)\s*$`)
	checkFieldPattern       = regexp.MustCompile(`^\s+-\s+([A-Za-z][A-Za-z ]*):\s*(.*?)\s*$`)
	checkIDPattern          = regexp.MustCompile(`^check-[a-z0-9][a-z0-9-]*$`)
	flowAcceptanceIDPattern = regexp.MustCompile(`\bA\d+\b`)
	flowDeclarationPattern  = regexp.MustCompile(`(?s)[\p{L}\p{N}_-]+\s*@?\s*(?:\[[\s\S]*?\]|\([\s\S]*?\)|\{[\s\S]*?\})`)
	flowReferencePattern    = regexp.MustCompile(`(?m)^\s*[\p{L}\p{N}_-]+\s*(?:(?:--|==|-\.)|(?:::[\p{L}\p{N}_-]+)?\s*(?:;|$))`)
	flowDirectivePattern    = regexp.MustCompile(`^\s*(?:flowchart|subgraph|end|direction|classDef|class|style|linkStyle|click)\b`)
	flowchartPattern        = regexp.MustCompile(`^\s*flowchart\s+(TD|LR|BT|RL)\s*$`)
	flowItemPattern         = regexp.MustCompile(`(?m)^\s*-\s+\[[ xX]\]\s+\x60?(` + acceptanceIDExpression + `)\x60?:`)
)

type executableCheck struct {
	id     string
	fields map[string]string
}

func extractAcceptanceIDs(body string) []string {
	acceptance := sectionContent(body, 2, "Acceptance")
	matches := acceptanceIDPattern.FindAllStringSubmatch(acceptance, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	return ids
}

func validateAcceptanceIDFormat(file, body string, report *diagnostic.Report) {
	acceptance := sectionContent(body, 2, "Acceptance")
	for _, match := range acceptanceTokenPattern.FindAllStringSubmatch(acceptance, -1) {
		if !canonicalAcceptanceID.MatchString(match[1]) {
			report.AddError(file, "invalid Acceptance ID: "+match[1]+"; expected A1, A2, ...")
		}
	}
}

func validateAcceptanceFlow(file, body string, report *diagnostic.Report) {
	acceptance, hasAcceptance := section(body, 2, "Acceptance")
	if !hasAcceptance {
		return
	}
	flow, hasFlow := section(acceptance.Content, 3, "Acceptance Flow")
	if !hasFlow {
		report.AddError(file, "Acceptance missing subsection: Acceptance Flow")
		return
	}

	lines := strings.Split(strings.ReplaceAll(flow.Content, "\r\n", "\n"), "\n")
	fence := strings.Repeat(string(rune(96)), 3)
	start := -1
	end := -1
	for index, line := range lines {
		if line == fence+"mermaid" && start < 0 {
			start = index
			continue
		}
		if start >= 0 && index > start && line == fence {
			end = index
			break
		}
	}
	if start < 0 {
		report.AddError(file, "Acceptance Flow must contain a Mermaid block")
		return
	}

	graphEnd := len(lines)
	if end >= 0 {
		graphEnd = end
	} else {
		report.AddError(file, "Acceptance Flow Mermaid block is not closed")
	}
	graph := lines[start+1 : graphEnd]
	statements := make([]string, 0, len(graph))
	hasFlowchart := false
	for _, line := range graph {
		if strings.HasPrefix(strings.TrimSpace(line), "%%") {
			continue
		}
		if flowchartPattern.MatchString(line) {
			hasFlowchart = true
		}
		if !flowDirectivePattern.MatchString(line) {
			statements = append(statements, line)
		}
	}
	if !hasFlowchart {
		report.AddError(file, "Acceptance Flow Mermaid block must declare a flowchart")
	}

	statementText := strings.Join(statements, "\n")
	if !flowDeclarationPattern.MatchString(statementText) && !flowReferencePattern.MatchString(statementText) {
		report.AddError(file, "Acceptance Flow must contain flow nodes")
	}
	flowIDs := make(map[string]struct{})
	for _, id := range flowAcceptanceIDPattern.FindAllString(statementText, -1) {
		flowIDs[id] = struct{}{}
	}
	if len(flowIDs) == 0 {
		report.AddError(file, "Acceptance Flow must reference at least one Acceptance ID")
	}
	expectedIDs := extractFlowAcceptanceIDs(acceptance.Content)
	expectedSet := make(map[string]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		expectedSet[id] = struct{}{}
		if _, ok := flowIDs[id]; !ok {
			report.AddError(file, "Acceptance Flow does not reference Acceptance ID: "+id)
		}
	}
	for id := range flowIDs {
		if _, ok := expectedSet[id]; !ok {
			report.AddError(file, "Acceptance Flow references unknown Acceptance ID: "+id)
		}
	}
}

func extractFlowAcceptanceIDs(acceptance string) []string {
	matches := flowItemPattern.FindAllStringSubmatch(acceptance, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	return ids
}

func validateExecutableChecks(file, body string, report *diagnostic.Report) {
	acceptance := sectionContent(body, 2, "Acceptance")
	references := readAcceptanceCheckReferences(file, acceptance, report)
	checksSection, hasChecks := section(acceptance, 3, "Executable Checks")
	if !hasChecks {
		if len(references) > 0 {
			report.AddError(file, "Acceptance references executable checks but has no Executable Checks section")
		}
		return
	}
	if !specdoc.Meaningful(checksSection.Content) {
		report.AddError(file, "Executable Checks section is empty")
		return
	}

	checks := readExecutableChecks(file, checksSection.Content, report)
	known := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		if _, duplicate := known[check.id]; duplicate {
			report.AddError(file, "Executable Checks has duplicate check: "+check.id)
		}
		known[check.id] = struct{}{}
		for _, field := range []string{"Run", "Environment", "Expected"} {
			if !specdoc.Meaningful(check.fields[field]) || specdoc.IsExplicitNone(check.fields[field]) {
				report.AddError(file, "Executable Check "+check.id+" missing "+field)
			}
		}
	}
	for _, checkID := range sortedMapKeys(references) {
		if _, found := known[checkID]; !found {
			report.AddError(file, "Acceptance references missing executable check: "+checkID)
		}
	}
	for _, checkID := range sortedMapKeys(known) {
		if _, referenced := references[checkID]; !referenced {
			report.AddError(file, "Executable Check is not referenced by an Acceptance item: "+checkID)
		}
	}
}

func readAcceptanceCheckReferences(file, acceptance string, report *diagnostic.Report) map[string]struct{} {
	references := make(map[string]struct{})
	currentAcceptance := ""
	acceptanceWithChecks := make(map[string]struct{})
	for _, line := range strings.Split(strings.ReplaceAll(acceptance, "\r\n", "\n"), "\n") {
		if match := acceptanceItemPattern.FindStringSubmatch(line); match != nil {
			currentAcceptance = match[1]
			continue
		}
		match := acceptanceChecksPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if currentAcceptance == "" {
			report.AddError(file, "Checks must belong to an Acceptance item")
			continue
		}
		if _, duplicate := acceptanceWithChecks[currentAcceptance]; duplicate {
			report.AddError(file, "Acceptance "+currentAcceptance+" has duplicate Checks field")
		}
		acceptanceWithChecks[currentAcceptance] = struct{}{}
		seen := make(map[string]struct{})
		checkIDs := parseCheckIDs(match[1])
		if len(checkIDs) == 0 {
			report.AddError(file, "Acceptance "+currentAcceptance+" Checks must reference at least one executable check")
		}
		for _, checkID := range checkIDs {
			if !checkIDPattern.MatchString(checkID) {
				report.AddError(file, "Acceptance "+currentAcceptance+" has invalid executable check id: "+checkID)
				continue
			}
			if _, duplicate := seen[checkID]; duplicate {
				report.AddError(file, "Acceptance "+currentAcceptance+" has duplicate executable check: "+checkID)
				continue
			}
			seen[checkID] = struct{}{}
			references[checkID] = struct{}{}
		}
	}
	return references
}

func readExecutableChecks(file, content string, report *diagnostic.Report) []executableCheck {
	checks := make([]executableCheck, 0)
	current := -1
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if match := checkStartPattern.FindStringSubmatch(line); match != nil {
			if !checkIDPattern.MatchString(match[1]) {
				report.AddError(file, "Executable Checks has invalid check id: "+match[1])
			}
			checks = append(checks, executableCheck{id: match[1], fields: map[string]string{}})
			current = len(checks) - 1
			continue
		}
		if match := checkFieldPattern.FindStringSubmatch(line); match != nil && current >= 0 {
			field := match[1]
			if field != "Run" && field != "Environment" && field != "Expected" {
				report.AddError(file, "Executable Check "+checks[current].id+" has unsupported field: "+field)
				continue
			}
			if _, duplicate := checks[current].fields[field]; duplicate {
				report.AddError(file, "Executable Check "+checks[current].id+" has duplicate field: "+field)
			}
			checks[current].fields[field] = match[2]
		}
	}
	if len(checks) == 0 {
		report.AddError(file, "Executable Checks must contain check blocks")
	}
	return checks
}

func parseCheckIDs(value string) []string {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(text, "[")
	text = strings.TrimSuffix(text, "]")
	parts := strings.Split(text, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.Trim(strings.TrimSpace(part), "`\"'")
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func validateAcceptancePolicy(file string, frontmatter specdoc.Frontmatter, report *diagnostic.Report) AcceptancePolicy {
	policy := AcceptanceHuman
	if value, found := frontmatter.String("acceptance_policy"); found {
		policy = AcceptancePolicy(value)
	} else if frontmatter.Has("acceptance_policy") {
		policy = ""
	}

	if policy != AcceptanceHuman && policy != AcceptanceVerified {
		report.AddError(file, "acceptance_policy must be human or verified")
		return AcceptanceHuman
	}
	verifiedFields := []string{"acceptance_authority", "acceptance_scope", "acceptance_review"}
	for _, field := range verifiedFields {
		value, found := frontmatter.String(field)
		if policy == AcceptanceVerified && !hasText(value, found) {
			report.AddError(file, "verified policy requires "+field)
		}
		if policy == AcceptanceHuman && frontmatter.Has(field) {
			report.AddError(file, field+" is only valid with verified policy")
		}
	}
	return policy
}

func validateOutcomeAcceptance(file string, frontmatter specdoc.Frontmatter, policy AcceptancePolicy, report *diagnostic.Report) AcceptancePolicy {
	method := AcceptanceHuman
	if value, found := frontmatter.String("acceptance_method"); found {
		method = AcceptancePolicy(value)
	} else if frontmatter.Has("acceptance_method") {
		method = ""
	}
	if method != AcceptanceHuman && method != AcceptanceVerified {
		report.AddError(file, "acceptance_method must be human or verified")
	}
	if method == AcceptanceVerified && policy != AcceptanceVerified {
		report.AddError(file, "verified acceptance requires Story acceptance_policy: verified")
	}
	acceptedBy, found := frontmatter.String("accepted_by")
	if !hasText(acceptedBy, found) {
		report.AddError(file, "accepted_by must identify the actual accepting person or executor")
	}
	return method
}
