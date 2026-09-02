package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	root:=t.TempDir(); os.Mkdir(filepath.Join(root,".git"),0755); os.WriteFile(filepath.Join(root,"package.json"),[]byte("{}"),0644); os.WriteFile(filepath.Join(root,"playwright.config.ts"),[]byte(""),0644)
	caps:=Detect(root); seen:=map[string]bool{}; for _,c:=range caps { seen[c.Name]=true }
	if !seen["git"] || !seen["javascript/typescript"] || !seen["playwright"] { t.Fatalf("unexpected capabilities: %+v",caps) }
}
