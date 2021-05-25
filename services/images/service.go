package images

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

type ImagesService struct {
	Store store.Store
}

func (is *ImagesService) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error) {
	images, metadata, err := is.Store.Images.GetAllPublic(filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return images, metadata, nil
}

func (is *ImagesService) ListForGallery(ctx context.Context, public bool, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	authData := store.ContextGetAuth(ctx)

	gallery, err := is.Store.Galleries.Get(galleryID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return nil, filters.Meta{}, store.ErrRecordNotFound
		default:
			return nil, filters.Meta{}, err
		}
	}

	if public {
		if !gallery.Published {
			return nil, filters.Meta{}, store.ErrForbidden
		}
	} else {
		if authData.User.ID != gallery.UserID {
			return nil, filters.Meta{}, store.ErrForbidden
		}
	}

	images, metadata, err := is.Store.Images.GetAllForGallery(galleryID, filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}

	return images, metadata, nil
}

func (is *ImagesService) Get(ctx context.Context, public bool, imageID int64) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	image, err := is.Store.Images.Get(imageID)
	if err != nil {
		return store.Image{}, err
	}

	if public {
		if !image.Published {
			return store.Image{}, store.ErrForbidden
		}
	} else {
		if authData.User.ID != image.UserID {
			return store.Image{}, store.ErrForbidden
		}
	}

	return image, nil
}

func (is *ImagesService) Download(ctx context.Context, public bool, imageID int64) (store.Image, io.ReadCloser, error) {
	image, err := is.Store.Images.Get(imageID)
	if err != nil {
		return store.Image{}, nil, err
	}

	if public {
		if !image.Published {
			return store.Image{}, nil, store.ErrForbidden
		}
	} else {
		authData := store.ContextGetAuth(ctx)
		if authData.User.ID != image.UserID {
			return store.Image{}, nil, store.ErrForbidden
		}
	}

	readCloser, err := is.Store.Images.GetReader(imageID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, nil, store.ErrEditConflict
		default:
			return store.Image{}, nil, err
		}
	}

	return image, readCloser, nil
}

func (is *ImagesService) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	gallery, err := is.Store.Galleries.Get(image.GalleryID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrRecordNotFound
		default:
			return store.Image{}, err
		}
	}

	if gallery.UserID != authData.User.ID {
		return store.Image{}, store.ErrForbidden
	}

	image, err = is.Store.Images.Insert(reader, store.Image{
		GalleryID:   image.GalleryID,
		Title:       image.Title,
		Caption:     image.Caption,
		UserID:      authData.User.ID,
		ContentType: image.ContentType,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "image_to_galleries_fk"):
			return store.Image{}, store.ErrEditConflict
		case errors.Is(err, store.ErrEmptyBytes):
			v := validator.New()
			v.AddError("body", "no bytes in body")
			return store.Image{}, v
		default:
			return store.Image{}, err
		}
	}

	return image, nil
}

func (is *ImagesService) Update(ctx context.Context, image store.Image) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	oldImage, err := is.Store.Images.Get(image.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrRecordNotFound
		default:
			return store.Image{}, err
		}
	}

	if oldImage.UserID != authData.User.ID {
		return store.Image{}, store.ErrForbidden
	}

	oldImage.Title = image.Title
	oldImage.Caption = image.Caption

	image, err = is.Store.Images.Update(oldImage)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "image_to_galleries_fk"):
			return store.Image{}, store.ErrEditConflict
		default:
			return store.Image{}, err
		}
	}

	return image, nil
}

func (is *ImagesService) Delete(ctx context.Context, imageID int64) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	image, err := is.Store.Images.Get(imageID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrRecordNotFound
		default:
			return store.Image{}, err
		}
	}

	if image.UserID != authData.User.ID {
		return store.Image{}, store.ErrForbidden
	}

	err = is.Store.Images.Delete(imageID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrRecordNotFound
		default:
			return store.Image{}, err
		}
	}

	return image, nil
}
