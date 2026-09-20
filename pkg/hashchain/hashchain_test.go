package hashchain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// TEST ASLI
// ============================================================================

func TestCanonicalJSON(t *testing.T) {
	assert.Equal(t, "{}", CanonicalJSON(nil))
	assert.Equal(t, "{}", CanonicalJSON(map[string]interface{}{}))

	m := map[string]interface{}{
		"zeta":   1,
		"alpha":  "first",
		"middle": true,
	}
	expected := `{"alpha":"first","middle":true,"zeta":1}`
	assert.Equal(t, expected, CanonicalJSON(m))

	nestedMap := map[string]interface{}{
		"user": map[string]interface{}{
			"role":  "signer",
			"email": "user@wesign.id",
			"details": map[string]interface{}{
				"zip":  "12345",
				"city": "Jakarta",
			},
		},
		"tags": []interface{}{"legal", "finance"},
	}
	expectedNested := `{"tags":["legal","finance"],"user":{"details":{"city":"Jakarta","zip":"12345"},"email":"user@wesign.id","role":"signer"}}`
	assert.Equal(t, expectedNested, CanonicalJSON(nestedMap))
}

func TestComputeHash_Consistency(t *testing.T) {
	actorID := uuid.New()
	docID := uuid.New()

	entry := AuditEntry{
		SequenceNumber: 1,
		EventType:      "document_uploaded",
		ActorID:        &actorID,
		DocumentID:     &docID,
		Metadata: map[string]interface{}{
			"file_name": "kontrak.pdf",
			"file_size": 1024,
		},
		PreviousHash: GenesisHash,
	}

	hash1 := ComputeHash(entry)
	hash2 := ComputeHash(entry)

	assert.NotEmpty(t, hash1)
	assert.Equal(t, hash1, hash2, "ComputeHash must be completely deterministic")
}

func TestVerifyChain_ValidChain(t *testing.T) {
	actorID := uuid.New()
	docID := uuid.New()

	e1 := AuditEntry{
		SequenceNumber: 1,
		EventType:      "document_uploaded",
		ActorID:        &actorID,
		DocumentID:     &docID,
		Metadata:       map[string]interface{}{"action": "upload"},
		PreviousHash:   GenesisHash,
	}
	e1.CurrentHash = ComputeHash(e1)

	e2 := AuditEntry{
		SequenceNumber: 2,
		EventType:      "signature_placed",
		ActorID:        &actorID,
		DocumentID:     &docID,
		Metadata:       map[string]interface{}{"page": 1},
		PreviousHash:   e1.CurrentHash,
	}
	e2.CurrentHash = ComputeHash(e2)

	e3 := AuditEntry{
		SequenceNumber: 3,
		EventType:      "document_sealed",
		ActorID:        nil,
		DocumentID:     &docID,
		Metadata:       map[string]interface{}{"pades": true},
		PreviousHash:   e2.CurrentHash,
	}
	e3.CurrentHash = ComputeHash(e3)

	chain := []AuditEntry{e1, e2, e3}

	valid, brokenIdx, err := VerifyChain(chain)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, -1, brokenIdx)
}

func TestVerifyChain_TamperDetection(t *testing.T) {
	createChain := func() (AuditEntry, AuditEntry) {
		actorID := uuid.New()
		docID := uuid.New()

		e1 := AuditEntry{
			SequenceNumber: 1,
			EventType:      "document_uploaded",
			ActorID:        &actorID,
			DocumentID:     &docID,
			Metadata:       map[string]interface{}{"amount": 1000},
			PreviousHash:   GenesisHash,
		}
		e1.CurrentHash = ComputeHash(e1)

		e2 := AuditEntry{
			SequenceNumber: 2,
			EventType:      "signature_placed",
			ActorID:        &actorID,
			DocumentID:     &docID,
			Metadata:       map[string]interface{}{"page": 1},
			PreviousHash:   e1.CurrentHash,
		}
		e2.CurrentHash = ComputeHash(e2)

		return e1, e2
	}

	t.Run("detects modified metadata content", func(t *testing.T) {
		e1, e2 := createChain()
		tamperedChain := []AuditEntry{e1, e2}
		tamperedChain[0].Metadata["amount"] = 999999

		valid, brokenIdx, err := VerifyChain(tamperedChain)
		assert.False(t, valid)
		assert.Equal(t, 0, brokenIdx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entry corrupted at index 0")
	})

	t.Run("detects broken previous hash linkage", func(t *testing.T) {
		e1, e2 := createChain()
		tamperedChain := []AuditEntry{e1, e2}
		tamperedChain[1].PreviousHash = "broken_previous_hash_value"

		valid, brokenIdx, err := VerifyChain(tamperedChain)
		assert.False(t, valid)
		assert.Equal(t, 1, brokenIdx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "previous hash mismatch")
	})

	t.Run("detects skipped sequence number", func(t *testing.T) {
		e1, e2 := createChain()
		e3Skipped := AuditEntry{
			SequenceNumber: 5,
			EventType:      "document_sealed",
			PreviousHash:   e2.CurrentHash,
			Metadata:       map[string]interface{}{},
		}
		e3Skipped.CurrentHash = ComputeHash(e3Skipped)

		chain := []AuditEntry{e1, e2, e3Skipped}
		valid, brokenIdx, err := VerifyChain(chain)
		assert.False(t, valid)
		assert.Equal(t, 2, brokenIdx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "chain sequence broken at index 2")
	})

	t.Run("empty chain is valid", func(t *testing.T) {
		valid, brokenIdx, err := VerifyChain(nil)
		assert.True(t, valid)
		assert.Equal(t, -1, brokenIdx)
		assert.NoError(t, err)
	})
}

// ============================================================================
// REGRESSION TEST — Perilaku setelah fix
// ============================================================================

// Entri pertama dengan PreviousHash bukan GenesisHash HARUS ditolak.
func TestRegression_HashChain_RejectsNonGenesisFirstEntry(t *testing.T) {
	actorID := uuid.New()

	rogue := AuditEntry{
		SequenceNumber: 1,
		EventType:      "document_uploaded",
		ActorID:        &actorID,
		Metadata:       map[string]interface{}{"action": "rogue"},
		PreviousHash:   "attacker-controlled-previous-hash",
	}
	rogue.CurrentHash = ComputeHash(rogue)

	second := AuditEntry{
		SequenceNumber: 2,
		EventType:      "signature_placed",
		ActorID:        &actorID,
		Metadata:       map[string]interface{}{"page": 1},
		PreviousHash:   rogue.CurrentHash,
	}
	second.CurrentHash = ComputeHash(second)

	chain := []AuditEntry{rogue, second}
	valid, brokenIdx, err := VerifyChain(chain)

	assert.False(t, valid, "chain harus ditolak karena tidak dimulai dari GenesisHash")
	assert.Equal(t, 0, brokenIdx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "first entry must link to GenesisHash")
}

// Entri pertama dengan SequenceNumber != 1 HARUS ditolak.
func TestRegression_HashChain_RejectsNonOneStartingSequence(t *testing.T) {
	actorID := uuid.New()

	rogue := AuditEntry{
		SequenceNumber: 99,
		EventType:      "document_uploaded",
		ActorID:        &actorID,
		Metadata:       map[string]interface{}{"action": "rogue"},
		PreviousHash:   GenesisHash,
	}
	rogue.CurrentHash = ComputeHash(rogue)

	second := AuditEntry{
		SequenceNumber: 100,
		EventType:      "signature_placed",
		ActorID:        &actorID,
		Metadata:       map[string]interface{}{"page": 1},
		PreviousHash:   rogue.CurrentHash,
	}
	second.CurrentHash = ComputeHash(second)

	chain := []AuditEntry{rogue, second}
	valid, brokenIdx, err := VerifyChain(chain)

	assert.False(t, valid)
	assert.Equal(t, 0, brokenIdx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "first entry sequence must be 1")
}

// CanonicalJSON tidak boleh meng-escape karakter HTML.
func TestRegression_CanonicalJSON_DoesNotEscapeHTML(t *testing.T) {
	m := map[string]interface{}{
		"text": "a<b>c&d",
	}
	result := CanonicalJSON(m)

	assert.NotContains(t, result, `\u003c`)
	assert.NotContains(t, result, `\u003e`)
	assert.NotContains(t, result, `\u0026`)

	rfcExpected := `{"text":"a<b>c&d"}`
	assert.Equal(t, rfcExpected, result,
		"CanonicalJSON harus selaras RFC 8785 (tidak escape HTML)")
}

// CanonicalJSON harus menghasilkan "null" untuk nilai yang tidak bisa di-marshal.
func TestRegression_CanonicalJSON_NullOnUnmarshalable(t *testing.T) {
	m := map[string]interface{}{
		"ch": make(chan int),
	}
	result := CanonicalJSON(m)

	assert.Equal(t, `{"ch":null}`, result,
		"nilai yang tidak bisa di-marshal harus menjadi null, bukan JSON malformed")
}
