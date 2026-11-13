package service

import "context"

type CatalogRepository interface {
	CreateFragment(context.Context, string, string, string, string, *int, *int, *int64, *int64) (CreateFragmentRecord, error)
	CreateMaterial(context.Context, string, string, string, string, string) (CreateMaterialRecord, error)
	GetMaterial(context.Context, string) (GetMaterialRecord, error)
	GetPublishedFragment(context.Context, string) (GetPublishedFragmentRecord, error)
	ListMaterials(context.Context) ([]ListMaterialsRecord, error)
	ListPublishedFragments(context.Context) ([]ListPublishedFragmentsRecord, error)
}
