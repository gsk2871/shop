// Package filestaging protects uploads across runtime generations during cleanup.
package filestaging

import (
	"path/filepath"
	"sync"
)

var active sync.Map

func canonical(path string) string {
	absolute, e := filepath.Abs(path)
	if e != nil {
		return path
	}
	parent, e := filepath.EvalSymlinks(filepath.Dir(absolute))
	if e != nil {
		return absolute
	}
	return filepath.Join(parent, filepath.Base(absolute))
}
func Protect(path string) func() {
	key := canonical(path)
	active.Store(key, struct{}{})
	return func() { active.Delete(key) }
}
func Active(path string) bool { _, ok := active.Load(canonical(path)); return ok }
