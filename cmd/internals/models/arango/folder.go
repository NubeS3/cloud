package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	"time"
)

type Folder struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Fullpath string  `json:"fullpath"`
	Children []Child `json:"children"`
}

type Child struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func InsertBucketFolder(bucketName string) (*Folder, error) {
	doc := &Folder{
		Name:     bucketName,
		Fullpath: bucketName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	meta, err := fileMetadataCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	doc.Id = meta.Key

	return doc, nil
}

func InsertFolder(name, parentId string) (*Folder, error) {
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	meta, err := fileMetadataCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	doc.Id = meta.Key
	_, err = AppendChildToFolderById(parent.Id, doc.Id, doc.Name, "folder")
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DocumentNotFound,
		}
	}

	return doc, nil
}

func InsertFile(name, parentId string) (*Folder, error) {
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	meta, err := fileMetadataCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	doc.Id = meta.Key
	_, err = AppendChildToFolderById(parent.Id, doc.Id, doc.Name, "file")
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DocumentNotFound,
		}
	}

	return doc, nil
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

	target, err = UpdateFullPath(targetId, to.Fullpath)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	to, err = RemoveChildOfFolder(to.Id, Child{
		Id:   target.Id,
		Name: target.Name,
		Type: target.Fullpath,
	})
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	app
}

func UpdateFullPath(id, newParentPath string) (*Folder, error) {
	if []rune(newParentPath)[len(newParentPath)-1] != '/' {
		newParentPath = newParentPath + "/"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR fol IN folders FILTER fol._key == @id " +
		"UPDATE fol WITH { fullpath: @newParentPath + fol.name } IN fol RETURN NEW"
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

func RemoveChildOfFolder(id string, child Child) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR f IN folders FILTER f._key == @id UPDATE f WITH { children: REMOVE_VALUE(doc.children, @child, 1)} RETURN NEW"
	bindVars := map[string]interface{}{
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var data Folder
	meta, err := fileMetadataCol.ReadDocument(ctx, id, &data)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
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

func AppendChildToFolderById(id, childId, childName, t string) (*Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR fol IN folders FILTER fol._key == @id " +
		"UPDATE fol WITH { children: APPEND(doc.children, @new } IN fol RETURN NEW"
	bindVars := map[string]interface{}{
		"id": id,
		"new": Child{
			Id:   childId,
			Name: childName,
			Type: t,
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
