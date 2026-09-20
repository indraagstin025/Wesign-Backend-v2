package hashchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// GenesisHash adalah hash awal untuk entri log pertama dalam rantai.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// AuditEntry merepresentasikan satu baris log audit sebelum dan sesudah di-hash.
type AuditEntry struct {
	SequenceNumber   int64                  `json:"sequence_number"`
	EventType        string                 `json:"event_type"`
	ActorID          *uuid.UUID             `json:"actor_id,omitempty"`
	DocumentID       *uuid.UUID             `json:"document_id,omitempty"`
	SigningRequestID *uuid.UUID             `json:"signing_request_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
	PreviousHash     string                 `json:"previous_hash"`
	CurrentHash      string                 `json:"current_hash"`
}

// ComputeHash menghitung SHA-256 dari entry sesuai formula audit trail WeSign.
func ComputeHash(e AuditEntry) string {
	h := sha256.New()

	h.Write([]byte(e.PreviousHash))
	h.Write(fmt.Appendf(nil, "%020d", e.SequenceNumber))
	h.Write([]byte(e.EventType))
	h.Write([]byte(uuidOrNull(e.ActorID)))
	h.Write([]byte(uuidOrNull(e.DocumentID)))
	h.Write([]byte(uuidOrNull(e.SigningRequestID)))
	h.Write([]byte(CanonicalJSON(e.Metadata)))

	return hex.EncodeToString(h.Sum(nil))
}

// CanonicalJSON menghasilkan representasi JSON kanonikal dengan kunci terurut
// alfabetis dan rekursif tanpa spasi, selaras RFC 8785.
func CanonicalJSON(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}
	return canonicalMap(m)
}

func canonicalMap(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := []byte("{")
	for i, k := range keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyJSON, err := marshalNoHTMLEscape(k)
		if err != nil {
			keyJSON = []byte(`""`)
		}
		buf = append(buf, keyJSON...)
		buf = append(buf, ':')
		buf = append(buf, canonicalValue(m[k])...)
	}
	buf = append(buf, '}')
	return string(buf)
}

func canonicalValue(v interface{}) []byte {
	switch val := v.(type) {
	case map[string]interface{}:
		return []byte(canonicalMap(val))
	case []interface{}:
		buf := []byte("[")
		for i, item := range val {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, canonicalValue(item)...)
		}
		buf = append(buf, ']')
		return buf
	default:
		res, err := marshalNoHTMLEscape(v)
		if err != nil {
			return []byte("null")
		}
		return res
	}
}

// marshalNoHTMLEscape melakukan json.Marshal tanpa meng-escape <, >, &
// agar selaras dengan RFC 8785.
func marshalNoHTMLEscape(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// VerifyChain memeriksa integritas urutan rantai audit log.
func VerifyChain(entries []AuditEntry) (bool, int, error) {
	if len(entries) == 0 {
		return true, -1, nil
	}

	var prev *AuditEntry
	for i := range entries {
		entry := &entries[i]

		if err := validateLinkage(prev, *entry, i); err != nil {
			return false, i, err
		}

		computed := ComputeHash(*entry)
		if computed != entry.CurrentHash {
			return false, i, fmt.Errorf("entry corrupted at index %d: computed hash %s != current hash %s",
				i, computed, entry.CurrentHash)
		}

		prev = entry
	}

	return true, -1, nil
}

func validateLinkage(prev *AuditEntry, curr AuditEntry, index int) error {
	if prev == nil {
		if curr.PreviousHash != GenesisHash {
			return fmt.Errorf("chain integrity violation at index 0: first entry must link to GenesisHash, got %q",
				curr.PreviousHash)
		}
		if curr.SequenceNumber != 1 {
			return fmt.Errorf("chain integrity violation at index 0: first entry sequence must be 1, got %d",
				curr.SequenceNumber)
		}
		return nil
	}

	if curr.PreviousHash != prev.CurrentHash {
		return fmt.Errorf("chain broken at index %d: previous hash mismatch", index)
	}
	if curr.SequenceNumber != prev.SequenceNumber+1 {
		return fmt.Errorf("chain sequence broken at index %d: expected %d, got %d",
			index, prev.SequenceNumber+1, curr.SequenceNumber)
	}

	return nil
}

func uuidOrNull(u *uuid.UUID) string {
	if u == nil {
		return "null"
	}
	return u.String()
}
