//
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package tag

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/iomz/go-llrp"
)

// ManagerService provides thread-safe tag management operations including adding,
// deleting, and retrieving tags. It maintains the tag collection and notifies
// connected clients when tags are updated.
type ManagerService struct {
	tags           llrp.Tags
	tagManagerChan chan Manager
	tagUpdatedChan chan llrp.Tags
	isConnAlive    *atomic.Bool
	mu             sync.Mutex
}

// NewManagerService creates and initializes a new tag manager service.
//
// Parameters:
//   - tagManagerChan: Channel for receiving tag management commands and sending responses
//   - tagUpdatedChan: Channel for notifying about tag updates (only when connection is alive)
//   - isConnAlive: Atomic boolean flag indicating whether an LLRP connection is active
func NewManagerService(tagManagerChan chan Manager, tagUpdatedChan chan llrp.Tags, isConnAlive *atomic.Bool) *ManagerService {

	var datasetTags llrp.Tags

	file, err := os.Open("tags.csv")
	if err == nil {

		reader := csv.NewReader(file)
		rows, _ := reader.ReadAll()

		fmt.Println("Loading tags from dataset...")

		for i, row := range rows {

			epc := row[1]

			epcBytes, err := hex.DecodeString(epc)
			if err != nil {
				fmt.Println("Invalid EPC:", epc)
				continue
			}

			tag := &llrp.Tag{
				EPC: epcBytes,
			}

			datasetTags = append(datasetTags, tag)

			fmt.Printf("Dataset Tag %d EPC: %s\n", i+1, epc)
		}

		file.Close()

	} else {

		fmt.Println("Dataset not found. Using default tags.")

		datasetTags = llrp.Tags{
			&llrp.Tag{EPC: []byte{0x30, 0x72, 0x27, 0x62, 0x7f, 0x2e, 0xa4, 0x80, 0x00, 0x00, 0x1c, 0x6a}},
			&llrp.Tag{EPC: []byte{0x30, 0x72, 0x27, 0x62, 0x7f, 0x2e, 0xa4, 0x80, 0x00, 0x00, 0x1c, 0x6b}},
			&llrp.Tag{EPC: []byte{0x30, 0x72, 0x27, 0x62, 0x7f, 0x2e, 0xa4, 0x80, 0x00, 0x00, 0x1c, 0x6c}},
		}
	}

	return &ManagerService{
		tags:           datasetTags,
		tagManagerChan: tagManagerChan,
		tagUpdatedChan: tagUpdatedChan,
		isConnAlive:    isConnAlive,
	}
}

// Process executes a tag management command (add, delete, or retrieve).
// It performs the operation thread-safely and sends the result back through
// the tagManagerChan. If tags are added or deleted and a connection is alive,
// it also notifies through tagUpdatedChan.
//
// The function releases the mutex before sending to channels to avoid deadlocks.
func (s *ManagerService) Process(cmd Manager) {
	var tagsToNotify llrp.Tags
	var shouldNotify bool

	s.mu.Lock()
	res := []*llrp.Tag{}
	switch cmd.Action {
	case AddTags:
		for _, t := range cmd.Tags {
			if i := s.tags.GetIndexOf(t); i < 0 {
				s.tags = append(s.tags, t)
				res = append(res, t)

				// Print detected tag EPC
				fmt.Printf("Detected Tag EPC: %X\n", t.EPC)
			}
		}
		if len(res) > 0 && s.isConnAlive.Load() {
			// Make a copy of tags before releasing the lock
			tagsToNotify = make(llrp.Tags, len(s.tags))
			copy(tagsToNotify, s.tags)
			shouldNotify = true
		}
	case DeleteTags:
		// Collect tags to keep
		toKeep := make(llrp.Tags, 0, len(s.tags))
		for _, t := range cmd.Tags {
			if i := s.tags.GetIndexOf(t); i >= 0 {
				res = append(res, t)
			}
		}
		// Rebuild tags excluding deleted ones
		for _, tag := range s.tags {
			if !slices.Contains(res, tag) {
				toKeep = append(toKeep, tag)
			}
		}
		s.tags = toKeep
		if len(res) > 0 && s.isConnAlive.Load() {
			// Make a copy of tags before releasing the lock
			tagsToNotify = make(llrp.Tags, len(s.tags))
			copy(tagsToNotify, s.tags)
			shouldNotify = true
		}
	case RetrieveTags:
		res = make(llrp.Tags, len(s.tags))
		copy(res, s.tags)

		fmt.Println("----------- Detected RFID Tags -----------")

		for i, tag := range res {
			fmt.Printf("Tag %d EPC: %X\n", i+1, tag.EPC)
		}

		fmt.Println("------------------------------------------")

		// Print each tag EPC separately
		fmt.Println("Detected Tags:")
		for i, tag := range res {
			fmt.Printf("Tag %d EPC: %X\n", i+1, tag.EPC)
		}
	}
	cmd.Tags = res
	s.mu.Unlock()

	// Send to channels without holding the lock to avoid deadlock
	if shouldNotify {
		s.tagUpdatedChan <- tagsToNotify
	}
	s.tagManagerChan <- cmd
}

// GetTags returns a copy of the current tag collection.
// The operation is thread-safe.
func (s *ManagerService) GetTags() llrp.Tags {
	s.mu.Lock()
	defer s.mu.Unlock()

	tags := make(llrp.Tags, len(s.tags))
	copy(tags, s.tags)

	fmt.Println("\nSERVER RFID TAGS FROM DATASET:")

	for i, t := range tags {
		fmt.Printf("Tag %d EPC: %X\n", i+1, t.EPC)
	}

	fmt.Println("------------------------------")

	return tags
}

// SetTags replaces the current tag collection with the provided tags.
// The operation is thread-safe.
func (s *ManagerService) SetTags(tags llrp.Tags) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tags = tags
}
