package database

import (
	"context"

	"github.com/augustin-wien/augustina-backend/ent"
	entcomment "github.com/augustin-wien/augustina-backend/ent/comment"
	entvendor "github.com/augustin-wien/augustina-backend/ent/vendor"
)

func (db *Database) GetVendorComments(vendorId int) (comments []*ent.Comment, err error) {
	ctx := context.Background()
	// Get vendor data
	comments, err = db.EntClient.Comment.Query().Where(entcomment.HasVendorWith(entvendor.ID(vendorId))).All(ctx)
	if err != nil {
		log.Error("GetVendorComments", err)
		return nil, err
	}
	return comments, nil

}

func (db *Database) CreateVendorComment(vendorID int, comment ent.Comment) (err error) {
	_, err = db.EntClient.Comment.Create().SetVendorID(vendorID).SetComment(comment.Comment).SetCreatedAt(comment.CreatedAt).SetResolvedAt(comment.ResolvedAt).SetWarning(comment.Warning).Save(context.Background())
	if err != nil {
		log.Error("CreateVendorComment", err)
	}
	return err
}

func (db *Database) UpdateVendorComment(vendorID int, comment ent.Comment) (err error) {
	_, err = db.EntClient.Comment.UpdateOneID(comment.ID).SetComment(comment.Comment).SetResolvedAt(comment.ResolvedAt).SetWarning(comment.Warning).Save(context.Background())
	if err != nil {
		log.Error("UpdateVendorComment", err)
	}
	return err
}

func (db *Database) DeleteVendorComment(vendorID int, commentID int) (err error) {
	err = db.EntClient.Comment.DeleteOneID(commentID).Exec(context.Background())
	if err != nil {
		log.Error("DeleteVendorComment", err)
	}
	return err
}
