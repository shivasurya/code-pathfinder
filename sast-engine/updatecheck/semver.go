package updatecheck

import (
	"fmt"
	"strconv"
	"strings"
)

// parseSemver parses a strict MAJOR.MINOR.PATCH version string.
// Returns the three components and ok=true on success.
// Returns ok=false (and zeros) for any non-conforming input.
func parseSemver(v string) (major, minor, patch int, ok bool) {
	parts := strings.SplitN(v, ".", 4)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var err error
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, 0, false
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, 0, false
	}
	if patch, err = strconv.Atoi(parts[2]); err != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

// Compare compares two MAJOR.MINOR.PATCH version strings.
// Returns 1 if a > b, -1 if a < b, 0 if a == b or either is malformed.
func Compare(a, b string) int {
	am, amin, ap, aok := parseSemver(a)
	bm, bmin, bp, bok := parseSemver(b)
	if !aok || !bok {
		return 0
	}
	switch {
	case am != bm:
		if am > bm {
			return 1
		}
		return -1
	case amin != bmin:
		if amin > bmin {
			return 1
		}
		return -1
	case ap != bp:
		if ap > bp {
			return 1
		}
		return -1
	default:
		return 0
	}
}

// Match reports whether version satisfies rangeExpr.
//
// Supported forms (subset of npm semver):
//
//	<X.Y.Z    strictly less than
//	<=X.Y.Z   less than or equal
//	>X.Y.Z    strictly greater than
//	>=X.Y.Z   greater than or equal
//	=X.Y.Z    exact match
//	A B       space-separated AND of exactly two constraints
//
// Wildcards, tildes, carets, and OR ranges are not supported.
// Returns (false, error) for any malformed input; callers should treat an
// error as "does not match" and skip the announcement silently.
func Match(rangeExpr, version string) (bool, error) {
	parts := strings.Fields(rangeExpr)
	switch len(parts) {
	case 0:
		return false, fmt.Errorf("updatecheck: empty range expression")
	case 1:
		return matchConstraint(parts[0], version)
	case 2:
		a, err := matchConstraint(parts[0], version)
		if err != nil {
			return false, err
		}
		b, err := matchConstraint(parts[1], version)
		if err != nil {
			return false, err
		}
		return a && b, nil
	default:
		return false, fmt.Errorf("updatecheck: range %q has too many constraints (max 2)", rangeExpr)
	}
}

// matchConstraint evaluates a single constraint (e.g. ">=1.2.3") against version.
func matchConstraint(constraint, version string) (bool, error) {
	var op, ver string
	switch {
	case strings.HasPrefix(constraint, "<="):
		op, ver = "<=", constraint[2:]
	case strings.HasPrefix(constraint, ">="):
		op, ver = ">=", constraint[2:]
	case strings.HasPrefix(constraint, "<"):
		op, ver = "<", constraint[1:]
	case strings.HasPrefix(constraint, ">"):
		op, ver = ">", constraint[1:]
	case strings.HasPrefix(constraint, "="):
		op, ver = "=", constraint[1:]
	default:
		return false, fmt.Errorf("updatecheck: constraint %q has no recognised operator", constraint)
	}

	if _, _, _, ok := parseSemver(ver); !ok {
		return false, fmt.Errorf("updatecheck: constraint %q contains invalid version %q", constraint, ver)
	}
	if _, _, _, ok := parseSemver(version); !ok {
		return false, fmt.Errorf("updatecheck: current version %q is not valid semver", version)
	}

	cmp := Compare(version, ver)
	switch op {
	case "<":
		return cmp < 0, nil
	case "<=":
		return cmp <= 0, nil
	case ">":
		return cmp > 0, nil
	case ">=":
		return cmp >= 0, nil
	default: // "="
		return cmp == 0, nil
	}
}
