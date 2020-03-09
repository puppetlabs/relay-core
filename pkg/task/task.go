package task

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

type Hash [sha1.Size]byte

func (h Hash) HexEncoding() string {
	return hex.EncodeToString(h[:])
}

func HashFromName(name string) Hash {
	return Hash(sha1.Sum([]byte(name)))
}

func HashFromID(id string) (h Hash, err error) {
	data, err := hex.DecodeString(id)
	if err != nil {
		return
	} else if len(data) != len(h) {
		err = fmt.Errorf("invalid hash")
		return
	}

	copy(h[:], data)
	return
}

// Metadata represents task metadata (such as the hash uniquely identifying the
// task).
type Metadata struct {
	Run  string
	Hash Hash
}

type Task struct {
	Run  string
	Name string
}

func (t *Task) TaskHash() Hash {
	taskName := fmt.Sprintf("run-%s-task-%s", t.Run, t.Name)
	return HashFromName(taskName)
}

func (t *Task) TaskMetadata() *Metadata {
	return &Metadata{
		Run:  t.Run,
		Hash: t.TaskHash(),
	}
}
