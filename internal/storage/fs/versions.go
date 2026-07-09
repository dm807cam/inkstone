package fs

import (
	"io"
	"path"
	"time"

	"github.com/ddvk/rmfakecloud/internal/storage"
	"github.com/ddvk/rmfakecloud/internal/storage/models"
	log "github.com/sirupsen/logrus"
)

// ListDocVersions reconstructs a document's version history from the account's
// root-modification log (.root.history). Blobs are never garbage-collected, so
// every generation in which the document's content hash changed is a restorable
// version. Returned newest-first.
func (fs *FileSystemStorage) ListDocVersions(uid, docid string) ([]storage.DocVersion, error) {
	historyPath := path.Join(fs.getUserBlobPath(uid), historyFile)
	history, err := models.ReadRootHistory(historyPath)
	if err != nil {
		return nil, err
	}

	ls := fs.BlobStorage(uid)

	versions := make([]storage.DocVersion, 0)
	lastHash := ""
	for _, h := range history {
		entry, err := h.DocEntry(ls, docid)
		if err != nil {
			// A single unreadable historical root shouldn't abort the whole list.
			log.Warnf("version history: skipping generation %d (%s): %v", h.Generation, h.Hash, err)
			continue
		}
		// Skip generations where the document was absent or unchanged.
		if entry == nil || entry.Hash == lastHash {
			continue
		}
		lastHash = entry.Hash
		versions = append(versions, storage.DocVersion{
			RootHash:   h.Hash,
			Generation: h.Generation,
			Date:       h.Date,
			Size:       entry.Size,
		})
	}

	// Newest first.
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}
	return versions, nil
}

// ExportVersion renders a specific historical version of a document to PDF.
// versionID is the root hash returned by ListDocVersions.
func (fs *FileSystemStorage) ExportVersion(uid, docid, versionID string) (io.ReadCloser, error) {
	ls := fs.BlobStorage(uid)
	histTree, err := (&models.RootHistory{Hash: versionID}).GetHashTree(ls)
	if err != nil {
		return nil, err
	}
	doc, err := histTree.FindDoc(docid)
	if err != nil {
		return nil, err
	}
	return fs.exportBlobDocument(doc, ls)
}

// RestoreVersion reverts a document's content to an earlier version, keeping it
// in its current location and bumping its version so devices re-sync. The old
// blobs are still present, so this re-points the document at them and commits a
// new generation.
func (fs *FileSystemStorage) RestoreVersion(uid, docid, versionID string) error {
	ls := fs.BlobStorage(uid)

	histTree, err := (&models.RootHistory{Hash: versionID}).GetHashTree(ls)
	if err != nil {
		return err
	}
	histDoc, err := histTree.FindDoc(docid)
	if err != nil {
		return err
	}

	tree, err := fs.GetCachedTree(uid)
	if err != nil {
		return err
	}

	return UpdateTree(tree, ls, func(t *models.HashTree) error {
		// Preserve the document's current location and bump its version; fall
		// back to the historical values if it is no longer in the tree.
		parent := histDoc.Parent
		version := 1
		exists := false
		if cur, err := t.FindDoc(docid); err == nil {
			parent = cur.Parent
			version = cur.Version + 1
			exists = true
		}

		meta := histDoc.MetadataFile
		meta.Parent = parent
		meta.Version = version
		meta.Deleted = false
		meta.LastModified = models.FromTime(time.Now())

		newDoc := models.NewHashDocWithMeta(docid, meta)
		newDoc.PayloadType = histDoc.PayloadType
		newDoc.Size = histDoc.Size
		// Deep-copy the historical file entries so rewriting the metadata entry
		// below doesn't mutate the throwaway historical tree.
		for _, e := range histDoc.Files {
			cp := *e
			newDoc.Files = append(newDoc.Files, &cp)
		}

		// Rewrite the metadata blob to reflect the adjusted fields; MetadataReader
		// repoints the metadata entry at the new hash in place.
		metaHash, metaReader, err := newDoc.MetadataReader()
		if err != nil {
			return err
		}
		if err := ls.Write(metaHash, metaReader); err != nil {
			return err
		}

		if err := newDoc.Rehash(); err != nil {
			return err
		}
		idxReader, err := newDoc.IndexReader()
		if err != nil {
			return err
		}
		if err := ls.Write(newDoc.Hash, idxReader); err != nil {
			return err
		}

		if exists {
			if err := t.Remove(docid); err != nil {
				return err
			}
		}
		return t.Add(newDoc)
	})
}
