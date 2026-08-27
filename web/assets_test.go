package web_test

import (
	"testing"

	"t2-display-blinder/web"
)

func TestEmbeddedAssets(t *testing.T) {
	files := []string{"index.html", "style.css", "app.js"}

	for _, file := range files {
		data, err := web.ReadAppFile(file)
		if err != nil {
			t.Errorf("failed to read embedded file %s: %v", file, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("embedded file %s is empty", file)
		}
	}
}
