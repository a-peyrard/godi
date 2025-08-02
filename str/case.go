package str

import "strings"

// ToScreamingSnakeCase transforms a given string into screaming snake case format
func ToScreamingSnakeCase(in string) string {
	in = strings.TrimSpace(in)
	if len(in) == 0 {
		return in
	}

	sb := strings.Builder{}
	sb.Grow(len(in) + len(in)/3) // estimate space for underscores

	for i, b := range []byte(in) {
		shouldWrite := true
		needsSeparator := false

		switch {
		case 'a' <= b && b <= 'z':
			b -= 'a' - 'A' // convert to uppercase
		case 'A' <= b && b <= 'Z':
			needsSeparator = true
		case b == '_' || b == '-':
			shouldWrite = false
			needsSeparator = true
		case '0' <= b && b <= '9':
			needsSeparator = true
		}

		if i > 0 && needsSeparator {
			sb.WriteByte('_')
		}

		if shouldWrite {
			sb.WriteByte(b)
		}
	}

	return sb.String()
}
