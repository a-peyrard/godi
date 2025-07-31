package main

import (
	"github.com/rs/zerolog"
	"regexp"
	"strconv"
	"strings"
)

type ProviderAnnotation struct {
	logger      *zerolog.Logger
	description string
	properties  map[string]string
}

func (p ProviderAnnotation) Priority() (priority int, found bool) {
	if priorityStr, exists := p.properties["priority"]; exists {
		if priority, err := strconv.Atoi(priorityStr); err == nil {
			return priority, true
		} else {
			p.logger.Warn().Msgf("Error parsing priority property: %s, skipping it", priorityStr)
		}
	}
	return 0, false
}

func (p ProviderAnnotation) Named() (named string, found bool) {
	named, found = p.properties["named"]
	return named, found
}

var knownProperties = []string{"priority", "named"}

func (p ProviderAnnotation) UnknownProperties() []string {
	var unknown []string
	for key := range p.properties {
		if !contains(knownProperties, key) {
			unknown = append(unknown, key)
		}
	}
	return unknown
}

type ParameterAnnotation struct {
	logger     *zerolog.Logger
	properties map[string]string
}

func (a ParameterAnnotation) Named() (named string, found bool) {
	named, found = a.properties["named"]
	return named, found
}

func parseProviderAnnotation(logger *zerolog.Logger, docText string) ProviderAnnotation {
	lines := strings.Split(docText, "\n")

	var descriptionLines []string
	var providerLine string

	// separate @provider line from description
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, providerAnnotationTag) {
			providerLine = line
		} else if line != "" && !strings.HasPrefix(line, "@") {
			descriptionLines = append(descriptionLines, line)
		}
	}

	return ProviderAnnotation{
		logger:      logger,
		description: strings.TrimSpace(strings.Join(descriptionLines, "\n")),
		properties:  parseProperties(providerLine, providerAnnotationTag),
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

func parseInjectAnnotation(logger *zerolog.Logger, comment string) ParameterAnnotation {
	content := strings.TrimPrefix(comment, "//")
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, injectAnnotationTag) {
		return ParameterAnnotation{properties: make(map[string]string)}
	}

	return ParameterAnnotation{
		logger:     logger,
		properties: parseProperties(content, injectAnnotationTag),
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
