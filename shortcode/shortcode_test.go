package shortcode

import (
  "testing"
	"github.com/google/uuid"
)

func TestGenerateShortcode(t *testing.T) {
  t.Run("adds new shortcode to the cache", func(t *testing.T) {
    shortcodeCache := make(map[string]bool)
    uuid := uuid.NewString()

    shortcode := GenerateShortcode(uuid, &shortcodeCache)
    if len(shortcode) == 0 {
      t.Errorf("failed to generate shortcode")
    }

    if !shortcodeCache[shortcode] {
      t.Errorf("the generated shortcode wasn't added to the cache")
    }
  })
}
