package hargen

import (
	"fmt"
	"math/rand"
	"strings"
)

// JSONGenerator creates random JSON objects with dictionary words
type JSONGenerator struct {
	dict     *Dictionary
	maxDepth int
	maxNodes int
	rng      *rand.Rand
	fatMode  bool
}

// NewJSONGenerator creates a new JSON generator
func NewJSONGenerator(dict *Dictionary, maxDepth, maxNodes int, rng *rand.Rand) *JSONGenerator {
	if maxDepth == 0 {
		maxDepth = 3
	}
	if maxNodes == 0 {
		maxNodes = 10
	}

	return &JSONGenerator{
		dict:     dict,
		maxDepth: maxDepth,
		maxNodes: maxNodes,
		rng:      rng,
		fatMode:  false,
	}
}

// SetFatMode enables fat mode for generating huge objects
func (jg *JSONGenerator) SetFatMode(enabled bool) {
	jg.fatMode = enabled
}

// GenerateObject creates a random JSON object with dictionary words
func (jg *JSONGenerator) GenerateObject(depth int) map[string]interface{} {
	// at max depth, just create simple key-value pair
	if depth >= jg.maxDepth {
		return map[string]interface{}{
			jg.dict.RandomWord(jg.rng): jg.dict.RandomWord(jg.rng),
		}
	}

	// determine how many nodes at this level (1 to maxNodes)
	nodeCount := jg.rng.Intn(jg.maxNodes) + 1
	obj := make(map[string]interface{}, nodeCount)

	for i := 0; i < nodeCount; i++ {
		key := jg.dict.RandomWord(jg.rng)

		// 30% chance of nesting deeper if not at max depth
		if depth < jg.maxDepth-1 && jg.rng.Float32() < 0.3 {
			obj[key] = jg.GenerateObject(depth + 1)
		} else {
			obj[key] = jg.dict.RandomWord(jg.rng)
		}
	}

	return obj
}

// InjectTerm injects a specific term into a JSON object at a random location
// returns the json path where it was injected (e.g., "user.name")
func (jg *JSONGenerator) InjectTerm(obj map[string]interface{}, term string) string {
	if len(obj) == 0 {
		// empty object, just add the term
		key := jg.dict.RandomWord(jg.rng)
		obj[key] = term
		return key
	}

	// randomly choose to inject as key or value
	injectAsKey := jg.rng.Float32() < 0.5

	// get all keys
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}

	if injectAsKey {
		// replace a random key with the term
		// (keep the original value)
		oldKey := keys[jg.rng.Intn(len(keys))]
		value := obj[oldKey]
		delete(obj, oldKey)
		obj[term] = value
		return term
	}

	// inject as value
	targetKey := keys[jg.rng.Intn(len(keys))]

	// if the value is a nested object, recurse
	if nestedObj, ok := obj[targetKey].(map[string]interface{}); ok {
		nestedPath := jg.InjectTerm(nestedObj, term)
		return targetKey + "." + nestedPath
	}

	// replace the value with the term
	obj[targetKey] = term
	return targetKey
}

// InjectTermIntoNewObject creates a new object with the term injected
func (jg *JSONGenerator) InjectTermIntoNewObject(term string) (map[string]interface{}, string) {
	obj := jg.GenerateObject(0)
	path := jg.InjectTerm(obj, term)
	return obj, path
}

// GenerateArray creates a random JSON array with dictionary words or objects
func (jg *JSONGenerator) GenerateArray(depth int, length int) []interface{} {
	if length == 0 {
		length = jg.rng.Intn(5) + 1
	}

	arr := make([]interface{}, length)
	for i := 0; i < length; i++ {
		// 30% chance of nested object, 70% simple value
		if depth < jg.maxDepth && jg.rng.Float32() < 0.3 {
			arr[i] = jg.GenerateObject(depth + 1)
		} else {
			arr[i] = jg.dict.RandomWord(jg.rng)
		}
	}

	return arr
}

// PathToString converts a path slice to dot-notation string
func PathToString(path []string) string {
	return strings.Join(path, ".")
}

// String values to generate for common fields
var commonFieldValues = map[string][]string{
	"email":    {"user@example.com", "test@test.com", "admin@company.org"},
	"phone":    {"+1234567890", "555-1234", "+44 20 7946 0958"},
	"country":  {"US", "UK", "CA", "DE", "FR", "JP"},
	"currency": {"USD", "EUR", "GBP", "JPY"},
	"status":   {"active", "pending", "completed", "failed"},
}

// GenerateRealisticValue generates a realistic value for a field name
func (jg *JSONGenerator) GenerateRealisticValue(fieldName string) string {
	// check if we have common values for this field
	lowerField := strings.ToLower(fieldName)
	if values, ok := commonFieldValues[lowerField]; ok {
		return values[jg.rng.Intn(len(values))]
	}

	// default to random word
	return jg.dict.RandomWord(jg.rng)
}

// GenerateFatObject creates a huge JSON object (~100KB) with base64 blobs
func (jg *JSONGenerator) GenerateFatObject() map[string]interface{} {
	obj := map[string]interface{}{
		"status":    "success",
		"timestamp": "2025-01-01T00:00:00Z",
		"metadata":  jg.GenerateObject(2),
	}

	// add 3-5 large base64 blob fields (~25KB each)
	blobCount := jg.rng.Intn(3) + 3
	for i := 0; i < blobCount; i++ {
		blobKey := fmt.Sprintf("blob_%d", i)
		obj[blobKey] = jg.generateBase64Blob(25000) // 25KB base64 string
	}

	// add large arrays with nested objects
	arrayCount := jg.rng.Intn(3) + 2
	for i := 0; i < arrayCount; i++ {
		arrayKey := fmt.Sprintf("items_%d", i)
		obj[arrayKey] = jg.generateLargeArray(50) // 50 items
	}

	// add deeply nested object tree
	obj["nested_data"] = jg.generateDeepObject(5, 20) // depth 5, 20 nodes per level

	return obj
}

// generateBase64Blob creates a base64-encoded string of specified byte size
func (jg *JSONGenerator) generateBase64Blob(sizeBytes int) string {
	// base64 encoding is ~4/3 the size, so generate 3/4 of target
	rawSize := (sizeBytes * 3) / 4
	bytes := make([]byte, rawSize)
	jg.rng.Read(bytes)

	// encode to base64
	return fmt.Sprintf("data:image/png;base64,%s", encodeBase64(bytes))
}

// simple base64 encoding (using standard chars)
func encodeBase64(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	result.Grow(((len(data) + 2) / 3) * 4)

	for i := 0; i < len(data); i += 3 {
		b1 := data[i]
		b2 := byte(0)
		b3 := byte(0)

		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}

		result.WriteByte(base64Chars[b1>>2])
		result.WriteByte(base64Chars[((b1&0x03)<<4)|(b2>>4)])
		result.WriteByte(base64Chars[((b2&0x0f)<<2)|(b3>>6)])
		result.WriteByte(base64Chars[b3&0x3f])
	}

	return result.String()
}

// generateLargeArray creates an array with many nested objects
func (jg *JSONGenerator) generateLargeArray(count int) []interface{} {
	arr := make([]interface{}, count)
	for i := 0; i < count; i++ {
		arr[i] = jg.GenerateObject(2) // nested objects
	}
	return arr
}

// generateDeepObject creates a deeply nested object with many nodes
func (jg *JSONGenerator) generateDeepObject(depth, nodesPerLevel int) map[string]interface{} {
	if depth == 0 {
		return map[string]interface{}{
			jg.dict.RandomWord(jg.rng): jg.dict.RandomWord(jg.rng),
		}
	}

	obj := make(map[string]interface{}, nodesPerLevel)
	for i := 0; i < nodesPerLevel; i++ {
		key := jg.dict.RandomWord(jg.rng)
		if i%3 == 0 && depth > 1 {
			obj[key] = jg.generateDeepObject(depth-1, nodesPerLevel/2)
		} else {
			obj[key] = jg.dict.RandomWord(jg.rng)
		}
	}

	return obj
}

// GenerateRealisticObject creates a more realistic JSON object with common field patterns
func (jg *JSONGenerator) GenerateRealisticObject(pattern string) map[string]interface{} {
	switch pattern {
	case "user":
		return map[string]interface{}{
			"id":       fmt.Sprintf("%d", jg.rng.Intn(10000)),
			"username": jg.dict.RandomWord(jg.rng) + jg.dict.RandomWord(jg.rng),
			"email":    jg.dict.RandomWord(jg.rng) + "@example.com",
			"profile": map[string]interface{}{
				"firstName": jg.dict.RandomWord(jg.rng),
				"lastName":  jg.dict.RandomWord(jg.rng),
				"age":       jg.rng.Intn(80) + 18,
			},
		}
	case "product":
		return map[string]interface{}{
			"id":          fmt.Sprintf("prod-%d", jg.rng.Intn(1000)),
			"name":        jg.dict.RandomWord(jg.rng) + " " + jg.dict.RandomWord(jg.rng),
			"price":       jg.rng.Float64() * 1000,
			"category":    jg.dict.RandomWord(jg.rng),
			"inStock":     jg.rng.Float32() < 0.8,
			"description": strings.Join(jg.dict.RandomWords(10, jg.rng), " "),
		}
	case "api_response":
		return map[string]interface{}{
			"status":  "success",
			"message": strings.Join(jg.dict.RandomWords(5, jg.rng), " "),
			"data":    jg.GenerateObject(1),
		}
	default:
		return jg.GenerateObject(0)
	}
}
