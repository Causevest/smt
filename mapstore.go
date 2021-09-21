package smt

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"golang.org/x/crypto/sha3"
)

// MapStore is a key-value store.
type MapStore interface {
	Get(key []byte) ([]byte, error)     // Get gets the value for a key.
	Set(key []byte, value []byte) error // Set updates the value for a key.
	Delete(key []byte) error            // Delete deletes a key.
	Export() ([]byte, error)            // exports the map into a byte array
}

// InvalidKeyError is thrown when a key that does not exist is being accessed.
type InvalidKeyError struct {
	Key []byte
}

func (e *InvalidKeyError) Error() string {
	return fmt.Sprintf("invalid key: %x", e.Key)
}

// SimpleMap is a simple in-memory map.
type SimpleMap struct {
	m map[string][]byte
}

// NewSimpleMap creates a new empty SimpleMap.
func NewSimpleMap() *SimpleMap {
	return &SimpleMap{
		m: make(map[string][]byte),
	}
}

// Get gets the value for a key.
func (sm *SimpleMap) Get(key []byte) ([]byte, error) {
	if value, ok := sm.m[string(key)]; ok {
		return value, nil
	}
	return nil, &InvalidKeyError{Key: key}
}

// Set updates the value for a key.
func (sm *SimpleMap) Set(key []byte, value []byte) error {
	sm.m[string(key)] = value
	return nil
}

// Export dumps the map into a gob serial
func (sm *SimpleMap) Export() ([]byte, error) {
	serial, err := GobEncode(sm.m)
	return serial, err
}

// Gob is used for encoding internal state
// and data. Json.Marshal is used for
// p2p communciation and RPC
// Works just like json.Marashal
func GobEncode(v interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	err := gob.NewEncoder(b).Encode(v)
	return b.Bytes(), err
}

// Works just like json.Unmarashal
func GobDecode(b []byte, v interface{}) error {
	buf := new(bytes.Buffer)
	buf.Write(b)
	err := gob.NewDecoder(buf).Decode(v)
	return err
}

// Delete deletes a key.
func (sm *SimpleMap) Delete(key []byte) error {
	_, ok := sm.m[string(key)]
	if ok {
		delete(sm.m, string(key))
		return nil
	}
	return &InvalidKeyError{Key: key}
}

// makes a new smt using a merklemap
// and the sha3 hash function and returns it
func NewMerkleTrie() *SparseMerkleTree {

	smn := NewSimpleMap()
	smv := NewSimpleMap()

	trie := NewSparseMerkleTree(smn, smv, sha3.New256())

	return trie
}

// used to save the a Trie to statedb
// keeps the root and map serial together
type TrieWrap struct {
	Root        []byte
	NodesBytes  []byte
	ValuesBytes []byte
}

func ImportTrie(wrap *TrieWrap) (*SparseMerkleTree, error) {
	// takes the gob encoded map for an smt and returns the smt
	smn, smv, err := ImportMerkleMap(wrap.NodesBytes, wrap.ValuesBytes)
	if err != nil {
		return nil, err
	}

	return ImportSparseMerkleTree(smn, smv, sha3.New256(), wrap.Root), nil
}

func ExportTrie(trie *SparseMerkleTree) (*TrieWrap, error) {
	nodesBytes, err := trie.nodes.Export()
	if err != nil {
		return nil, err
	}

	valuesBytes, err := trie.values.Export()
	if err != nil {
		return nil, err
	}

	wrap := TrieWrap{
		Root:        trie.Root(),
		NodesBytes:  nodesBytes,
		ValuesBytes: valuesBytes,
	}
	return &wrap, nil
}

func ImportMerkleMap(nodesBytes, valuesBytes []byte) (*SimpleMap, *SimpleMap, error) {
	var smn, smv SimpleMap
	err := GobDecode(nodesBytes, &smn.m)
	if err != nil {
		return nil, nil, err
	}

	err = GobDecode(valuesBytes, &smv.m)
	if err != nil {
		return nil, nil, err
	}

	return &smn, &smv, err
}
