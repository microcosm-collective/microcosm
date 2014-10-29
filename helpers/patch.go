package helpers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/lib/pq"
)

type PatchType struct {
	Operation string         `json:"op"`
	Path      string         `json:"path"`
	From      string         `json:"from,omitempty"`
	RawValue  interface{}    `json:"value,omitempty"`
	Bool      sql.NullBool   `json:"-"`
	String    sql.NullString `json:"-"`
	Int64     sql.NullInt64  `json:"-"`
	Time      pq.NullTime    `json:"-"`
}

// Partially implements http://tools.ietf.org/html/rfc6902
//
// Patch examples:
// { "op": "test", "path": "/a/b/c", "value": "foo" },
// { "op": "remove", "path": "/a/b/c" },
// { "op": "add", "path": "/a/b/c", "value": [ "foo", "bar" ] },
// { "op": "replace", "path": "/a/b/c", "value": 42 },
// { "op": "move", "from": "/a/b/c", "path": "/a/b/d" },
// { "op": "copy", "from": "/a/b/d", "path": "/a/b/e" }
func TestPatch(patches []PatchType) (int, error) {
	if 0 == len(patches) {
		return http.StatusBadRequest, errors.New("Patch: no patches were provided")
	}

	for _, v := range patches {
		switch v.Operation {
		case "add":
			if strings.Trim(v.Path, " ") == "" || v.RawValue == nil {
				return http.StatusBadRequest, errors.New("Patch: add operation incorrectly specified")
			}
			return http.StatusNotImplemented, errors.New("Patch: json-patch 'add' operation not implemented")
		case "copy":
			if strings.Trim(v.Path, " ") == "" || strings.Trim(v.From, " ") == "" {
				return http.StatusBadRequest, errors.New("Patch: copy operation incorrectly specified")
			}
			return http.StatusNotImplemented, errors.New("Patch: json-patch 'copy' operation not implemented")
		case "move":
			if strings.Trim(v.Path, " ") == "" || strings.Trim(v.From, " ") == "" {
				return http.StatusBadRequest, errors.New("Patch: move operation incorrectly specified")
			}
			return http.StatusNotImplemented, errors.New("Patch: json-patch 'move' operation not implemented")
		case "remove":
			if strings.Trim(v.Path, " ") == "" {
				return http.StatusBadRequest, errors.New("Patch: remove operation incorrectly specified")
			}
			return http.StatusNotImplemented, errors.New("Patch: json-patch 'remove' operation not implemented")
		case "replace":
			if strings.Trim(v.Path, " ") == "" || v.RawValue == nil {
				return http.StatusBadRequest, errors.New("Patch: replace operation incorrectly specified")
			}
		case "test":
			if strings.Trim(v.Path, " ") == "" || v.RawValue == nil {
				return http.StatusBadRequest, errors.New("Patch: test operation incorrectly specified")
			}
			return http.StatusNotImplemented, errors.New("Patch: json-patch 'test' operation not implemented")
		default:
			return http.StatusBadRequest, errors.New("Patch: unsupported operation in patch")
		}
	}

	return http.StatusOK, nil
}

func (p *PatchType) ScanRawValue() (int, error) {

	switch p.RawValue.(type) {
	case bool:
		p.Bool = sql.NullBool{Bool: p.RawValue.(bool), Valid: true}
	case string:
		p.String = sql.NullString{String: p.RawValue.(string), Valid: true}
	default:
		return http.StatusNotImplemented, errors.New("Patch: Currently only values of type boolean and string patchable")
	}

	return http.StatusOK, nil
}
