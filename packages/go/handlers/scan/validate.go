// Author: Caden Lund
package scan

import (
	"fmt"
	"strings"

	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// maxPhotos is the most photos a scan takes, as scan.v1's Photo and docs/contracts.md state it.
// Each photo's own rules are in the handlers root, which plates share.
const maxPhotos = 3

// maxItems is the longest list a verdict keeps, from the model or from an edit. The tag on
// buildEdit's list restates it.
const maxItems = 40

// itemInput holds one ingredient, normalized, to app.scan_item's checks (keep the two in step),
// and to plain text on one line, which the table cannot hold a NUL against anyway.
type itemInput struct {
	Name     string `pfield:"name"     validate:"required,max=40,plaintext"`
	Category string `pfield:"category" validate:"required,oneof=protein vegetable fruit grain dairy condiment drink snack leftover other"`
	Quantity string `pfield:"quantity" validate:"max=20,plaintext"`
}

// buildPhotos checks a scan's photos and returns their bytes for the model. Every problem is
// refused on photos, the field the camera screen shows it under, naming the photo it is about.
func buildPhotos(in []*scanv1.Photo) ([][]byte, handlers.FieldErrors) {
	var invalid handlers.FieldErrors
	refuse := func(msg string) {
		invalid = append(invalid, handlers.FieldError{Field: "photos", Message: msg})
	}

	if len(in) == 0 || len(in) > maxPhotos {
		refuse(fmt.Sprintf("a scan takes 1 to %d photos", maxPhotos))
	}
	photos := make([][]byte, 0, len(in))
	for i, photo := range in {
		n := i + 1
		jpeg := photo.GetJpeg()
		switch {
		case !handlers.IsJPEG(jpeg):
			refuse(fmt.Sprintf("photo %d isn't a JPEG", n))
		case len(jpeg) > handlers.MaxPhotoBytes:
			refuse(fmt.Sprintf("photo %d is over 2 MB", n))
		}
		switch photo.GetKind() {
		case scanv1.PhotoKind_PHOTO_KIND_FRIDGE, scanv1.PhotoKind_PHOTO_KIND_FREEZER,
			scanv1.PhotoKind_PHOTO_KIND_PANTRY:
		case scanv1.PhotoKind_PHOTO_KIND_UNSPECIFIED:
			refuse(fmt.Sprintf("photo %d must be of the fridge, freezer or pantry", n))
		default:
			// A kind newer than this server: refused the same way, naming the ones it knows.
			refuse(fmt.Sprintf("photo %d must be of the fridge, freezer or pantry", n))
		}
		photos = append(photos, jpeg)
	}
	if invalid != nil {
		return nil, invalid
	}
	return photos, nil
}

// buildEdit normalizes and validates an edit: the scan's id and the whole corrected list. Every
// problem comes back at once, each under its ingredient's index (ingredients[2].name), so the
// sheet can mark them all. A list over maxItems is refused, never cut short.
func buildEdit(req *scanv1.EditIngredientsRequest) ([]repository.ScanItem, handlers.FieldErrors) {
	invalid := handlers.Validate(struct {
		ScanID      int64                `pfield:"scan_id"     validate:"gt=0"`
		Ingredients []*scanv1.Ingredient `pfield:"ingredients" validate:"max=40"`
	}{req.GetScanId(), req.GetIngredients()})
	if len(req.GetIngredients()) > maxItems {
		return nil, invalid
	}

	items := make([]repository.ScanItem, 0, len(req.GetIngredients()))
	for i, ingredient := range req.GetIngredients() {
		item, bad := buildItem(ingredient.GetName(), categoryWord(ingredient.GetCategory()),
			ingredient.GetQuantity())
		for _, fe := range bad {
			invalid = append(invalid, handlers.FieldError{
				Field:   fmt.Sprintf("ingredients[%d].%s", i, fe.Field),
				Message: fe.Message,
			})
		}
		items = append(items, item)
	}
	if invalid != nil {
		return nil, invalid
	}
	return items, nil
}

// keepModelItems holds the model's list to the same rules as a player's edit, since it is just as
// untrusted. An item that fails is dropped, as is anything past maxItems, and how many went comes
// back for the log.
func keepModelItems(in []vision.Item) ([]repository.ScanItem, int) {
	kept := make([]repository.ScanItem, 0, min(len(in), maxItems))
	for _, item := range in {
		if len(kept) == maxItems {
			break
		}
		category := strings.ToLower(strings.TrimSpace(item.Category))
		row, invalid := buildItem(item.Name, category, item.Quantity)
		if invalid != nil {
			continue
		}
		kept = append(kept, row)
	}
	return kept, len(in) - len(kept)
}

// buildItem normalizes one ingredient and holds it to app.scan_item's checks. category is the
// word the database stores.
func buildItem(name, category, quantity string) (repository.ScanItem, handlers.FieldErrors) {
	in := itemInput{
		Name:     strings.TrimSpace(name),
		Category: category,
		Quantity: strings.TrimSpace(quantity),
	}
	invalid := handlers.Validate(in)
	if invalid != nil {
		return repository.ScanItem{}, invalid
	}
	return repository.ScanItem{Name: in.Name, Category: in.Category, Quantity: in.Quantity}, nil
}
