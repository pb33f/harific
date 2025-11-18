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
	}
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
