package main

import (
	"fmt"
	"github.com/a-peyrard/godi/set"
	"github.com/rs/zerolog"
	"regexp"
	"strconv"
	"strings"
)

type (
	ProviderDecoratorAnnotation struct {
		logger      *zerolog.Logger
		description string
		properties  map[string]string

		conditions []WhenAnnotation
	}

	WhenAnnotation struct {
		logger   *zerolog.Logger
		named    string
		operator string
		value    string
	}
)

func (p ProviderDecoratorAnnotation) Priority() (priority int, found bool) {
	if priorityStr, exists := p.properties["priority"]; exists {
		if priority, err := strconv.Atoi(priorityStr); err == nil {
			return priority, true
		} else {
			p.logger.Warn().Msgf("Error parsing priority property: %s, skipping it", priorityStr)
		}
	}
	return 0, false
}

func (p ProviderDecoratorAnnotation) Named() (named string, found bool) {
	named, found = p.properties["named"]
	return named, found
}

var knownProperties = set.NewWithValues("priority", "named")

func (p ProviderDecoratorAnnotation) UnknownProperties() []string {
	var unknown []string
	for key := range p.properties {
		if knownProperties.DoesNotContain(key) {
			unknown = append(unknown, key)
		}
	}
	return unknown
}

func parseProviderDecoratorAnnotation(logger *zerolog.Logger, fnName string, docText string, providerOrDecoratorTag string) ProviderDecoratorAnnotation {
	lines := strings.Split(docText, "\n")

	var (
		descriptionLines []string
		providerLine     string
		conditionLines   []string
	)
	// separate @provider line, and @when lines from description
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, providerOrDecoratorTag) {
			providerLine = line
		} else if strings.HasPrefix(line, whenAnnotationTag) {
			conditionLines = append(conditionLines, line)
		} else if line != "" && !strings.HasPrefix(line, "@") {
			descriptionLines = append(descriptionLines, line)
		}
	}

	return ProviderDecoratorAnnotation{
		logger:      logger,
		description: formatDescription(fnName, descriptionLines),
		properties:  parseProperties(providerLine, providerOrDecoratorTag),
		conditions:  parseWhenAnnotations(logger, conditionLines),
	}
}

func parseProperties(line string, tag string) map[string]string {
	properties := make(map[string]string)

	if line == "" {
		return properties
	}

	// remove "@provider" prefix
	content := strings.TrimPrefix(line, tag)
	content = strings.TrimSpace(content)

	if content == "" {
		return properties
	}

	// regex to match key=value or key="value" patterns
	re := regexp.MustCompile(`(\w+)=(?:"([^"]*)"|(\w+))`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		key := match[1]
		// match[2] is quoted value, match[3] is unquoted value
		value := match[2]
		if value == "" {
			value = match[3]
		}
		properties[key] = value
	}

	return properties
}

type InjectAnnotation struct {
	logger     *zerolog.Logger
	properties map[string]string
}

func (a InjectAnnotation) String() string {
	return fmt.Sprintf("InjectAnnotation(\"%s\")", a.properties)
}

func (a InjectAnnotation) Named() (named string, found bool) {
	named, found = a.properties["named"]
	return named, found
}

func (a InjectAnnotation) Multiple() (multiple bool, found bool) {
	var raw string
	raw, found = a.properties["multiple"]
	if !found {
		return false, true
	}
	multiple, err := strconv.ParseBool(raw)
	if err != nil {
		a.logger.Warn().Err(err).Msg("Error parsing multiple, not a correct bool")
		return false, found
	}
	return multiple, found
}

func (a InjectAnnotation) Optional() (value bool, found bool) {
	optionalStr, found := a.properties["optional"]
	return optionalStr == "true", found
}

func parseInjectAnnotation(logger *zerolog.Logger, comment string) InjectAnnotation {
	content := strings.TrimPrefix(comment, "//")
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, injectAnnotationTag) {
		return InjectAnnotation{properties: make(map[string]string)}
	}

	return InjectAnnotation{
		logger:     logger,
		properties: parseProperties(content, injectAnnotationTag),
	}
}

type ConfigAnnotation struct {
	logger      *zerolog.Logger
	description string
	properties  map[string]string
}

func (a ConfigAnnotation) String() string {
	return fmt.Sprintf("ConfigAnnotation(\"%s\")", a.properties)
}

func (a ConfigAnnotation) Prefix() string {
	prefix, found := a.properties["prefix"]
	if !found {
		prefix = ""
	}
	return prefix
}

func parseConfigAnnotation(logger *zerolog.Logger, configType string, docText string) ConfigAnnotation {
	lines := strings.Split(docText, "\n")

	var (
		configLine       string
		descriptionLines []string
	)
	// separate @config line from description
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, configAnnotationTag) {
			configLine = line
		} else if line != "" && !strings.HasPrefix(line, "@") {
			descriptionLines = append(descriptionLines, line)
		}
	}

	// separate @config line from description
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, configAnnotationTag) {
			configLine = line
			break
		}
	}

	return ConfigAnnotation{
		logger:      logger,
		description: formatDescription(configType, descriptionLines),
		properties:  parseProperties(configLine, configAnnotationTag),
	}
}

func parseWhenAnnotations(logger *zerolog.Logger, lines []string) []WhenAnnotation {
	if len(lines) == 0 {
		return nil
	}
	conditions := make([]WhenAnnotation, 0, len(lines))
	for _, line := range lines {
		annotation, err := parseWhenAnnotation(logger, line)
		if err != nil {
			logger.Warn().Err(err).Msgf("Failed to parse @when annotation: %s", line)
			continue
		}
		conditions = append(conditions, annotation)
	}

	return conditions
}

func parseWhenAnnotation(logger *zerolog.Logger, line string) (WhenAnnotation, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, whenAnnotationTag) {
		return WhenAnnotation{}, fmt.Errorf("line does not start with %s: %s", whenAnnotationTag, line)
	}

	content := strings.TrimPrefix(line, whenAnnotationTag)
	content = strings.TrimSpace(content)

	if content == "" {
		return WhenAnnotation{}, fmt.Errorf("empty @when annotation")
	}

	properties := parseProperties(content, whenAnnotationTag)
	named, found := properties["named"]
	if !found {
		return WhenAnnotation{}, fmt.Errorf("missing 'named' property in @when annotation: %s", line)
	}
	valueEq, equalsFound := properties["equals"]
	valueNotEq, notEqualsFound := properties["not_equals"]
	if !equalsFound && !notEqualsFound {
		return WhenAnnotation{}, fmt.Errorf("missing 'equals' or 'not_equals' property in @when annotation: %s", line)
	}

	operator := "equals"
	if notEqualsFound {
		operator = "not_equals"
	}
	value := strings.TrimSpace(valueEq)
	if notEqualsFound {
		value = strings.TrimSpace(valueNotEq)
	}

	return WhenAnnotation{
		logger:   logger,
		named:    named,
		operator: operator,
		value:    value,
	}, nil
}

func formatDescription(typeStr string, descriptionLines []string) string {
	normalized := strings.TrimSpace(strings.Join(descriptionLines, "\n"))
	normalized = strings.TrimPrefix(normalized, typeStr)
	normalized = strings.TrimSpace(normalized)

	return normalized
}
