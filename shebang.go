package outline

import (
	"bytes"
	"path"
	"regexp"
	"strings"
)

var rubyInterpreter = regexp.MustCompile(`^ruby(?:[0-9]+(?:\.[0-9]+)*)?$`)

func rubyShebang(src []byte) bool {
	line, _, _ := bytes.Cut(src, []byte{'\n'})
	if !bytes.HasPrefix(line, []byte("#!")) {
		return false
	}
	fields := strings.Fields(string(line[2:]))
	if len(fields) == 0 || strings.HasSuffix(fields[0], "/") {
		return false
	}
	interpreter := fields[0]
	if path.Base(interpreter) == "env" {
		interpreter = envInterpreter(fields[1:])
	}
	return !strings.HasSuffix(interpreter, "/") && rubyInterpreter.MatchString(path.Base(interpreter))
}

func envInterpreter(fields []string) string {
	options := true
	for len(fields) != 0 {
		field := fields[0]
		fields = fields[1:]
		if !literalEnvWord(field) {
			return ""
		}
		if options {
			switch field {
			case "--":
				options = false
				continue
			case "-", "-i", "--ignore-environment", "-S", "--split-string":
				continue
			case "-u", "--unset", "-C", "--chdir":
				if len(fields) == 0 || !literalEnvWord(fields[0]) {
					return ""
				}
				fields = fields[1:]
				continue
			}
			if strings.HasPrefix(field, "--unset=") || strings.HasPrefix(field, "--chdir=") {
				continue
			}
			if split, ok := envSplitString(field); ok {
				fields = append([]string{split}, fields...)
				continue
			}
		}
		if strings.HasPrefix(field, "-") {
			return ""
		}
		if index := strings.IndexByte(field, '='); index > 0 {
			options = false
			continue
		}
		return field
	}
	return ""
}

func envSplitString(field string) (string, bool) {
	if value, ok := strings.CutPrefix(field, "--split-string="); ok {
		return value, true
	}
	return strings.CutPrefix(field, "-S")
}

func literalEnvWord(field string) bool {
	// Quoting, escapes and expansion require more than whitespace splitting.
	return field != "" && !strings.HasPrefix(field, "#") && !strings.ContainsAny(field, "'\"\\$`")
}
