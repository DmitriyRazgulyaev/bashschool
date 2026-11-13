package service

import (
	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) Materials(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListMaterials(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка материалов")
	}

	items := []domain.Object{}
	for _, row := range rows1 {
		var id, title, kind, license, body string
		var created any
		{
			id, title, kind, license, body, created =
				row.ID, row.Title, row.Kind, row.License, row.Body, row.Created

			items = append(items, domain.Object{
				"id":          id,
				"title":       title,
				"type":        kind,
				"license":     license,
				"description": body,
				"createdAt":   created,
			})
		}

	}

	return domain.OK(items), nil
}

func (s *Service) Fragments(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListPublishedFragments(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка фрагментов")
	}

	items := []domain.Object{}
	for _, row := range rows1 {
		var id, text, title, chapter string
		{
			id, text, title, chapter = row.ID, row.Text, row.Title, row.Chapter

			items = append(items, domain.Object{
				"id":      id,
				"textBa":  text,
				"source":  title,
				"chapter": chapter,
			})
		}

	}

	return domain.OK(items), nil
}

func (s *Service) AddMaterial(ctx context.Context, p domain.Parameters, in AddMaterialInput) (domain.Result, error) {
	var id string
	recordCreateMaterial, err := s.store.CreateMaterial(ctx, in.Title, in.Type, in.License, in.Description, in.SourceURL)
	id = recordCreateMaterial.ID

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось создать материал")
	}
	return domain.New(domain.Object{"id": id}), nil
}

func (s *Service) AddFragment(ctx context.Context, p domain.Parameters, in AddFragmentInput) (domain.Result, error) {
	if in.SourceLang == "" {
		in.SourceLang = "ba"
	}
	if !langPattern.MatchString(in.SourceLang) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_LANGUAGE", "Код языка должен содержать 2–3 латинские буквы")
	}
	var id string
	recordCreateFragment, err := s.store.CreateFragment(ctx, in.MaterialID, in.TextBa, in.SourceLang, in.Chapter, in.StartOffset, in.EndOffset, in.StartTimeMs, in.EndTimeMs)
	id = recordCreateFragment.ID

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_MATERIAL", "Не удалось создать фрагмент")
	}
	return domain.New(domain.Object{"id": id}), nil
}

func (s *Service) MaterialDetail(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var id, title, kind, license, description string
	recordGetMaterial, err := s.store.GetMaterial(ctx, p.ID)
	id, title, kind, license, description =
		recordGetMaterial.ID, recordGetMaterial.Title, recordGetMaterial.Kind,
		recordGetMaterial.License, recordGetMaterial.Description

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Материал не найден")
	}
	return domain.OK(domain.Object{
		"id":          id,
		"title":       title,
		"type":        kind,
		"license":     license,
		"description": description,
	}), nil
}

func (s *Service) FragmentDetail(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var id, text, title, chapter string
	var start, end *int
	var startMs, endMs *int64
	recordGetPublishedFragment, err := s.store.GetPublishedFragment(ctx, p.ID)
	id, text, title, chapter, start, end, startMs,
		endMs = recordGetPublishedFragment.ID, recordGetPublishedFragment.Text, recordGetPublishedFragment.Title, recordGetPublishedFragment.Chapter, recordGetPublishedFragment.Start,
		recordGetPublishedFragment.End, recordGetPublishedFragment.StartMs, recordGetPublishedFragment.EndMs

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Фрагмент не найден")
	}
	return domain.OK(domain.Object{"id": id, "textBa": text, "source": domain.Object{
		"title":       title,
		"chapter":     chapter,
		"startOffset": start,
		"endOffset":   end,
		"startTimeMs": startMs,
		"endTimeMs":   endMs,
	}}), nil
}
