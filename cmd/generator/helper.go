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

func parseProviderAnnotation(logger *zerolog.Logger, docText string) ProviderAnnotation {
	lines := strings.Split(docText, "\n")

	var descriptionLines []string
	var providerLine string

	// Separate @provider line from description
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "@provider") {
			providerLine = line
		} else if line != "" && !strings.HasPrefix(line, "@") {
			// Add non-empty, non-annotation lines to description
			descriptionLines = append(descriptionLines, line)
		}
	}

	// Clean up description - remove extra whitespace
	description := strings.TrimSpace(strings.Join(descriptionLines, "\n"))

	// Parse properties from @provider line
	properties := parseProviderProperties(providerLine)

	return ProviderAnnotation{
		logger,
		description,
		properties,
	}
}

func parseProviderProperties(providerLine string) map[string]string {
	properties := make(map[string]string)

	if providerLine == "" {
		return properties
	}

	// remove "@provider" prefix
	content := strings.TrimPrefix(providerLine, "@provider")
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
