package lens

import (
	"testing"
)

func TestReverseLens(t *testing.T) {
	t.Run("rename", func(t *testing.T) {
		originalLens := RenamePropertyOperation{
			op:          "rename",
			source:      "name",
			destination: "title",
		}

		expectedLens := RenamePropertyOperation{
			op:          "rename",
			source:      "title",
			destination: "name",
		}

		op := ReverseRenamePropertyOp(originalLens)

		if op != expectedLens {
			t.Errorf("got lens: %+v\nexpected: %+v", op, expectedLens)
		}
	})

	t.Run("add", func(t *testing.T) {
		originalLens := AddPropertyOperation{
			op:           "add",
			name:         "title",
			dataType:     "string",
			defaultValue: "Title",
			required:     true,
		}

		expectedLens := RemovePropertyOperation{
			op:   "remove",
      name: "title",
		}

		op := ReverseAddPropertyOp(originalLens)

		if op != expectedLens {
			t.Errorf("got lens: %+v\nexpected: %+v", op, expectedLens)
		}
	})
}
