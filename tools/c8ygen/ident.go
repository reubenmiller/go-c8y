package main

import (
	"sort"
	"strings"
	"unicode"
)

// commonInitialisms are upper-cased wholesale when they form a path/word segment,
// matching Go naming conventions and golint expectations.
var commonInitialisms = map[string]string{
	"id":    "ID",
	"url":   "URL",
	"uri":   "URI",
	"api":   "API",
	"http":  "HTTP",
	"https": "HTTPS",
	"json":  "JSON",
	"xml":   "XML",
	"sso":   "SSO",
	"tfa":   "TFA",
	"totp":  "TOTP",
	"crl":   "CRL",
	"est":   "EST",
	"ca":    "CA",
	"ui":    "UI",
	"oauth": "OAuth",
}

// pascal converts an arbitrary token into a PascalCase Go identifier fragment,
// applying the common-initialism rules.
func pascal(s string) string {
	if s == "" {
		return ""
	}
	if init, ok := commonInitialisms[strings.ToLower(s)]; ok {
		return init
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// pascalAll splits a token on word boundaries (non-alphanumerics and camelCase humps)
// and PascalCases each part, so "application_no_microservice_manifest" becomes
// "ApplicationNoMicroserviceManifest" and "grant_type" becomes "GrantType".
func pascalAll(s string) string {
	var b strings.Builder
	for _, tok := range splitToken(s) {
		b.WriteString(pascal(tok))
	}
	return b.String()
}

// splitToken breaks a raw path/word segment on any non-alphanumeric boundary and on
// camelCase humps, so "managedObjects" -> ["managed","Objects"] and "verify-cert" ->
// ["verify","cert"].
func splitToken(s string) []string {
	// First split on non-alphanumeric runes.
	rawFields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := []string{}
	for _, f := range rawFields {
		out = append(out, splitCamel(f)...)
	}
	return out
}

// splitCamel splits a camelCase/PascalCase word into its component words.
func splitCamel(s string) []string {
	var words []string
	var cur strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(runes[i-1]) {
			words = append(words, cur.String())
			cur.Reset()
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

// pathIdent derives a stable, exported Go identifier from an API path.
// "/alarm/alarms/{id}" -> "AlarmAlarmsID".
// "/inventory/managedObjects/{id}/childAdditions" -> "InventoryManagedObjectsIDChildAdditions".
func pathIdent(path string) string {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	var b strings.Builder
	for _, seg := range segs {
		// {id} -> Id, {childId} -> ChildId
		seg = strings.TrimSuffix(strings.TrimPrefix(seg, "{"), "}")
		b.WriteString(pascalAll(seg))
	}
	id := b.String()
	if id == "" {
		return "Root"
	}
	// Ensure it does not start with a digit.
	if unicode.IsDigit(rune(id[0])) {
		id = "X" + id
	}
	return id
}

// sortedKeys returns the keys of a map in deterministic order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
