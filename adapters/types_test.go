package adapters

import "testing"

func TestStorageQuotaExceededError_Error(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		err := &StorageQuotaExceededError{Message: "custom quota error"}
		if err.Error() != "custom quota error" {
			t.Errorf("expected 'custom quota error', got '%s'", err.Error())
		}
	})

	t.Run("with empty message", func(t *testing.T) {
		err := &StorageQuotaExceededError{}
		if err.Error() != "storage quota exceeded" {
			t.Errorf("expected 'storage quota exceeded', got '%s'", err.Error())
		}
	})
}
