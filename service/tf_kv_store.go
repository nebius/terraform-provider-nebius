package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type TfKeyValueStorage interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// GetObject retrieves the JSON-encoded value stored under the provided key,
// unmarshals it into the target object, returns a boolean indicating if the key
// was found, and any diagnostics encountered.
func GetObject(
	ctx context.Context,
	storage TfKeyValueStorage,
	key string,
	target any,
) (bool, diag.Diagnostics) {
	// Retrieve the raw JSON bytes from the storage.
	data, diags := storage.GetKey(ctx, key)
	if diags.HasError() {
		return false, diags
	}
	// Optionally: if data is empty, you might want to handle that separately (e.g., return a not-found error).
	if len(data) == 0 {
		// No data found. Depending on your use case, you might want to return an error diag.
		return false, nil
	}
	// Unmarshal the JSON data into the provided target.
	if err := json.Unmarshal(data, target); err != nil {
		return true, ErrorToDiag(
			diags,
			err,
			fmt.Sprintf("failed to unmarshal JSON for key '%s'", key),
		)
	}
	return true, nil
}

// SetObject marshals the given object to JSON and saves it into the storage under the specified key.
func SetObject(
	ctx context.Context,
	storage TfKeyValueStorage,
	key string,
	obj any,
) diag.Diagnostics {
	// Marshal the object into JSON.
	data, err := json.Marshal(obj)
	if err != nil {
		return ErrorToDiag(
			nil,
			err,
			fmt.Sprintf("failed to marshal object for key '%s'", key),
		)
	}
	// Save the JSON data in storage.
	return storage.SetKey(ctx, key, data)
}
