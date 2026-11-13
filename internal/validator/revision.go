package validator

import "fmt"

func validatePreparedRevision(prepared, current string) string {
	if !revisionPattern.MatchString(prepared) {
		return "spec-revision must match rev-N from nanospec prepare"
	}
	if prepared != current {
		return fmt.Sprintf("specification changed: prepared %s, current %s; rerun nanospec prepare and reassess the Story, dependencies and evidence before continuing", prepared, current)
	}
	return ""
}
