package arango

import (
	"context"
	"time"

	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/arangodb/go-driver"
)

type Folder struct {
	Id       string  `json:"-"`
	OwnerId  string  `json:"owner_id"`
	Name     string  `json:"name"`
	Fullpath string  `json:"fullpath"`
	Children []Child `json:"children"`
}

type Child struct {
	Id       string      `json:"id"`
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	IsHidden bool        `json:"is_hidden"`
	Metadata interface{} `json:"metadata"`
}

type ChildFileMetadata struct {
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

func InsertBucketFolder(bucketName string) (*Folder, error) {
	bucket, err := FindBucketByName(bucketName)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	doc := &Folder{
		Name:     bucketName,
		Fullpath: "/" + bucketName,
		OwnerId:  bucket.Uid,
		Children: []Child{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	meta, err := folderCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	doc.Id = meta.Key

	////LOG CREATE BUCKET FOLDER
	//_ = nats.SendFolderEvent(doc.Id, doc.OwnerId, doc.Name, doc.Fullpath, "create_root")

	return doc, nil
}

func InsertFolder(name, parentId, ownerId string) (*Folder, error) {
	parent, err := FindFolderById(parentId)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     "folder with id " + parentId + " is not found",
			ErrType: models.DocumentNotFound,
		}
	}

	doc := &Folder{
		Name:     name,
		Fullpath: parent.Fullpath + "/" + name,
		OwnerId:  ownerId,
		Children: []Child{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	meta, err := folderCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	doc.Id = meta.Key
	_, err = AppendChildToFolderById(parent.Id, Child{
		Id:       doc.Id,
		Name:     doc.Name,
		Type:     "folder",
		IsHidden: false,
	})

	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DocumentNotFound,
		}
	}

	//LOG CREATE BUCKET FOLDER
	//_ = nats.SendFolderEvent(doc.Id, doc.OwnerId, doc.Name, doc.Fullpath, "create")

	return doc, nil
}

func InsertFile(fid, fname, parentId, contentType string, size int64, isHidden bool) (*Folder, error) {
	f, err := AppendChildToFolderById(parentId, Child{
		Id:       fid,
		Name:     fname,
		Type:     "file",
		IsHidden: isHidden,
		Metadata: ChildFileMetadata{
			ContentType: contentType,
			Size:        size,
		},
	})
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DocumentNotFound,
		}
	}

	return f, nil
}

func InsertFileByPath(fid, fname, parentPath string) (*Folder, error) {
	f, err := AppendChildToFolderByPath(parentPath, Child{
		Id:   fid,
		Name: fname,
		Type: "file",
	})
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DocumentNotFound,
		}
	}

	return f, nil
}

func FindFolderByOwnerId(oid string, limit int64, offset int64) ([]Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR fol IN folders FILTER fol.owner_id == @oid LIMIT @offset, @limit RETURN fol"
	bindVars := map[string]interface{}{
		"oid":    oid,
		"offset": offset,
		"limit":  limit,
	}
	folders := []Folder{}
	folder := Folder{}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		folder.Id = meta.Key
		folders = append(folders, folder)
	}

	return folders, nil
}

func MoveFolderById(targetId string, toId string) (*Folder, error) {
	target, err := FindFolderById(targetId)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.NotFound,
		}
	}

	to, err := FindFolderById(targetId)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.NotFound,
		}
	}

	oldParentPath := ultis.GetParentPath(target.Fullpath)

	target, err = UpdateFullPath(targetId, to.Fullpath)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	_, _ = RemoveChildOfFolderByPath(oldParentPath, Child{
		Id:       target.Id,
		Name:     target.Name,
		Type:     "folder",
		IsHidden: false,
	})

	target, err = AppendChildToFolderById(to.Id, Child{
		Id:       target.Id,
		Name:     target.Name,
		Type:     "folder",
		IsHidden: false,
	})

	return target, nil
}

func UpdateFullPath(id, newParentPath string) (*Folder, error) {
	if []rune(newParentPath)[len(newParentPath)-1] != '/' {
		newParentPath = newParentPath + "/"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR fol IN folders FILTER fol._key == @id " +
		"UPDATE fol WITH { fullpath: @newParentPath + fol.name } IN folders RETURN NEW"
	bindVars := map[string]interface{}{
		"id":            id,
		"newParentPath": newParentPath,
	}

	folder := Folder{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return &folder, nil
}

func RemoveChildOfFolderByPath(path string, child Child) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR f IN folders FILTER f.fullpath == @path UPDATE f WITH { children: REMOVE_VALUE(f.children, @child, 1)} IN folders RETURN NEW"
	bindVars := map[string]interface{}{
		"path":  path,
		"child": child,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	folder := Folder{}
	for {
		meta, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		folder.Id = meta.Key
	}

	if folder.Id == "" {
		return nil, &models.ModelError{
			Msg:     "folder not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &folder, nil
}

func RemoveChildOfFolder(id string, child Child) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR f IN folders FILTER f._key == @id UPDATE f WITH { children: REMOVE_VALUE(doc.children, @child, 1)} IN folders RETURN NEW"
	bindVars := map[string]interface{}{
		"id":    id,
		"child": child,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	folder := Folder{}
	for {
		_, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if folder.Id == "" {
		return nil, &models.ModelError{
			Msg:     "folder not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &folder, nil
}

func FindFolderById(id string) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	var data Folder
	meta, err := folderCol.ReadDocument(ctx, id, &data)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DocumentNotFound,
		}
	}

	data.Id = meta.Key
	return &data, nil
}

func FindFolderByFullpath(fullpath string) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR f IN folders FILTER f.fullpath == @fullpath LIMIT 1 RETURN f"
	bindVars := map[string]interface{}{
		"fullpath": fullpath,
	}

	folder := Folder{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		folder.Id = meta.Key
	}

	if folder.Id == "" {
		return nil, &models.ModelError{
			Msg:     "folder not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &folder, nil
}

func AppendChildToFolderById(toId string, child Child) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR fol IN folders FILTER fol._key == @id " +
		"UPDATE fol WITH { children: APPEND(fol.children, @new) } IN folders RETURN NEW"
	bindVars := map[string]interface{}{
		"id":  toId,
		"new": child,
	}

	folder := Folder{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return &folder, nil
}

func AppendChildToFolderByPath(toPath string, child Child) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR fol IN folders FILTER fol.fullpath == @path " +
		"UPDATE fol WITH { children: APPEND(fol.children, @new) } IN folders RETURN NEW"
	bindVars := map[string]interface{}{
		"path": toPath,
		"new":  child,
	}

	folder := Folder{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return &folder, nil
}

func UpdateHiddenStatusOfFolderChild(path, fid, name string, hiddenStatus bool) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR f IN folders FILTER f.fullpath == @path LIMIT 1 " +
		"LET alterL = REPLACE_NTH(f.children, POSITION(f.children, { name: @childName }, true),  @newChild) " +
		"UPDATE f WITH { children: alterL } IN folders RETURN NEW"
	bindVars := map[string]interface{}{
		"path":      path,
		"childName": name,
		"newChild": Child{
			Id:       fid,
			Name:     name,
			Type:     "file",
			IsHidden: hiddenStatus,
		},
	}

	folder := Folder{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &folder)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return &folder, nil
}

func RemoveFolderAndItsChildren(parentPath, name string) error {
	fullpath := parentPath + "/" + name

	folder, err := FindFolderByFullpath(fullpath)
	if err != nil {
		return err
	}

	//Find bucket parent of the folder's children
	bucket, e := FindBucketByName(ultis.GetBucketName(fullpath))
	if e != nil {
		return e
	}

	//Remove all the folder's children
	for _, child := range folder.Children {
		if child.Type == "file" {
			err = MarkDeleteFile(fullpath, child.Name, bucket.Id)
		} else {
			err = RemoveFolderAndItsChildren(fullpath, child.Name)
		}
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR f IN folders FILTER f.fullpath == @fullpath REMOVE f IN folders LET removed = OLD RETURN removed"
	bindVars := map[string]interface{}{
		"fullpath": fullpath,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	//parentPath == "" means that the removed folder is a bucket, it does not have parent to call the next section
	if parentPath == "" {
		return nil
	}

	//Remove the folder from its parent
	_, err = RemoveChildOfFolderByPath(parentPath, Child{
		Id:       folder.Id,
		Name:     folder.Name,
		Type:     "folder",
		IsHidden: false,
	})
	if err != nil {
		return err
	}

	return nil
}
