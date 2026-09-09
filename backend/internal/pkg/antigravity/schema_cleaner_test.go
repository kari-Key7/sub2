package antigravity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanJSONSchema_NestedArrayItems(t *testing.T) {
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"where": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "array",
						},
					},
				},
			},
		},
	}

	cleaned := CleanJSONSchema(inputSchema)
	require.NotNil(t, cleaned)

	queryProps := cleaned["properties"].(map[string]any)["query"].(map[string]any)["properties"].(map[string]any)
	whereSchema := queryProps["where"].(map[string]any)
	require.Equal(t, "array", whereSchema["type"])

	itemsSchema, ok := whereSchema["items"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "array", itemsSchema["type"])

	innerItemsSchema, ok := itemsSchema["items"].(map[string]any)
	require.True(t, ok, "inner items must be present and a map")
	require.NotEmpty(t, innerItemsSchema["type"], "inner items must have type")
}
