package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/seaweedfs"
	"github.com/arangodb/go-driver"
	"time"
)

type SnapshotStatus int

type SnapshotTargetType int

const (
	Preparing SnapshotStatus = iota
	Processing
	Finish
	Error
)

const (
	FILE SnapshotTargetType = iota
	FOLDER
)

type Snapshot struct {
	Id         string           `json:"_key,omitempty"`
	Owner      string           `json:"owner"`
	Status     SnapshotStatus   `json:"status"`
	Target     []SnapshotTarget `json:"target"`
	Error      []SnapshotError  `json:"error"`
	SnapFileId string           `json:"snap_file_id,omitempty"`
}

type SnapshotTarget struct {
	TargetType SnapshotTargetType `json:"target_type"`
	Path       string             `json:"path"`
	Name       string             `json:"name"`
	Size       int64              `json:"size,omitempty"`
	FileId     string             `json:"file_id,omitempty"`
}

type SnapshotError struct {
	Error  string         `json:"error"`
	Target SnapshotTarget `json:"target"`
}

type SnapshotInput struct {
	TargetType SnapshotTargetType
	Id         string
}

func CreateSnapshot(targetList []SnapshotInput, owner string) (*Snapshot, error) {
	doc := Snapshot{
		Status: Preparing,
		Target: []SnapshotTarget{},
		Error:  []SnapshotError{},
		Owner:  owner,
	}

	for _, target := range targetList {
		if target.TargetType == FILE {
			file, err := FindMetadataById(target.Id)
			if err != nil {
				return nil, err
			}
			doc.Target = append(doc.Target, SnapshotTarget{
				TargetType: target.TargetType,
				Path:       file.Path,
				Name:       file.Name,
				Size:       file.Size,
				FileId:     file.FileId,
			})
		} else {
			folder, err := FindFolderById(target.Id)
			if err != nil {
				return nil, err
			}
			doc.Target = append(doc.Target, SnapshotTarget{
				TargetType: target.TargetType,
				Path:       folder.Fullpath,
				Name:       folder.Name,
			})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	meta, err := snapCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	doc.Id = meta.Key

	return &doc, nil
}

func FindSnapshotById(id string) (*Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	snap := Snapshot{}
	meta, err := snapCol.ReadDocument(ctx, id, &snapCol)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &models.ModelError{
				Msg:     "bucket not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	snap.Id = meta.Key

	return &snap, nil
}

func FindSnapshotsByOwner(uid string, limit, offset int) ([]Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR s IN snapshots FILTER s.owner == @uid LIMIT @offset, @limit RETURN s"
	bindVars := map[string]interface{}{
		"uid":    uid,
		"limit":  limit,
		"offset": offset,
	}

	snaps := []Snapshot{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		s := Snapshot{}
		meta, err := cursor.ReadDocument(ctx, &s)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		s.Id = meta.Key
		snaps = append(snaps, s)
	}

	return snaps, nil
}

func DeleteSnapshot(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	snap, err := FindSnapshotById(id)
	if err != nil {
		return &models.ModelError{
			Msg:     "snapshot not found",
			ErrType: models.DocumentNotFound,
		}
	}

	if snap.Status == Processing {
		return &models.ModelError{
			Msg:     "cannot delete processing snapshot",
			ErrType: models.Locked,
		}
	}

	_, err = snapCol.RemoveDocument(ctx, snap.Id)
	if err != nil {
		if driver.IsNotFound(err) {
			return &models.ModelError{
				Msg:     "snapshot not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	if snap.Status == Finish || snap.Status == Error {
		err = seaweedfs.DeleteFile(snap.SnapFileId)
		if err != nil {
			return err
		}
	}

	//LOG CREATE BUCKET
	//_ = nats.SendBucketEvent(bucket.Id, bucket.Uid, bucket.Name, bucket.Region, "delete")

	return nil
}
