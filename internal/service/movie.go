package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"homework/internal/model"
	"homework/internal/repository"
)

type MovieService struct {
	repository repository.MovieRepository
}

type ValidationError struct {
	Message string
}

func (validationError ValidationError) Error() string { return validationError.Message }

type NotFoundError struct {
	Message string
}

func (notFoundError NotFoundError) Error() string { return notFoundError.Message }

type InvalidIDError struct{}

func (InvalidIDError) Error() string { return "Invalid movie id" }

type InvalidPaginationError struct {
	Message string
}

func (paginationError InvalidPaginationError) Error() string { return paginationError.Message }

func NewMovieService(repository repository.MovieRepository) *MovieService {
	return &MovieService{repository: repository}
}

func ValidateMovieJSON(body []byte) (model.MovieInput, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		log.Printf("invalid movie JSON: %v", err)
		return model.MovieInput{}, ValidationError{Message: "Request body must be a JSON object"}
	}
	if fields == nil {
		return model.MovieInput{}, ValidationError{Message: "Request body must be a JSON object"}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			log.Printf("invalid trailing movie JSON: %v", err)
		}
		return model.MovieInput{}, ValidationError{Message: "Request body must contain one JSON object"}
	}

	var input model.MovieInput
	requiredFields := []struct {
		name   string
		target *string
	}{
		{name: "title", target: &input.Title},
		{name: "director", target: &input.Director},
		{name: "country", target: &input.Country},
		{name: "genre", target: &input.Genre},
	}
	for _, field := range requiredFields {
		raw, ok := fields[field.name]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return model.MovieInput{}, ValidationError{Message: field.name + " is required"}
		}
		if err := json.Unmarshal(raw, field.target); err != nil || strings.TrimSpace(*field.target) == "" {
			return model.MovieInput{}, ValidationError{Message: field.name + " must be a non-empty string"}
		}
	}

	if raw, ok := fields["release_year"]; ok && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		year, err := strconv.ParseInt(string(bytes.TrimSpace(raw)), 10, strconv.IntSize)
		if err != nil {
			return model.MovieInput{}, ValidationError{Message: "release_year must be an integer"}
		}
		value := int(year)
		input.ReleaseYear = &value
	}
	if raw, ok := fields["synopsis"]; ok && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		var synopsis string
		if err := json.Unmarshal(raw, &synopsis); err != nil {
			return model.MovieInput{}, ValidationError{Message: "synopsis must be a string"}
		}
		input.Synopsis = &synopsis
	}
	return input, nil
}

func (service *MovieService) Create(body []byte) (model.Movie, error) {
	input, err := ValidateMovieJSON(body)
	if err != nil {
		return model.Movie{}, err
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	return service.repository.Create(model.Movie{
		Title:       input.Title,
		Director:    input.Director,
		Country:     input.Country,
		Genre:       input.Genre,
		ReleaseYear: input.ReleaseYear,
		Synopsis:    input.Synopsis,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (service *MovieService) GetByID(id uint32) (model.Movie, bool) {
	return service.repository.GetByID(id)
}

func (service *MovieService) Replace(id uint32, body []byte) (model.Movie, error) {
	input, err := ValidateMovieJSON(body)
	if err != nil {
		return model.Movie{}, err
	}
	current, ok := service.repository.GetByID(id)
	if !ok {
		return model.Movie{}, NotFoundError{Message: "Movie not found"}
	}
	current.Title = input.Title
	current.Director = input.Director
	current.Country = input.Country
	current.Genre = input.Genre
	current.ReleaseYear = input.ReleaseYear
	current.Synopsis = input.Synopsis
	now := time.Now().UTC().Truncate(time.Microsecond)
	latestTimestamp := current.UpdatedAt
	if current.CreatedAt.After(latestTimestamp) {
		latestTimestamp = current.CreatedAt
	}
	// Ensure an immediate replacement still advances at the stored microsecond precision.
	if !now.After(latestTimestamp) {
		now = latestTimestamp.Add(time.Microsecond)
	}
	current.UpdatedAt = now
	updated, ok := service.repository.Update(current)
	if !ok {
		return model.Movie{}, NotFoundError{Message: "Movie not found"}
	}
	return updated, nil
}

func (service *MovieService) Delete(id uint32) error {
	if !service.repository.Delete(id) {
		return NotFoundError{Message: "Movie not found"}
	}
	return nil
}

func (service *MovieService) ListPage(filter, pageValue, limitValue string, hasPage, hasLimit bool) ([]model.Movie, error) {
	page, limit, err := parsePaginationValues(pageValue, limitValue, hasPage, hasLimit)
	if err != nil {
		return nil, err
	}

	total := service.repository.Count(filter)
	offset, empty := paginationOffset(page, limit, total)
	if empty {
		return make([]model.Movie, 0), nil
	}
	return service.repository.List(filter, offset, limit), nil
}

func parsePaginationValues(pageValue, limitValue string, hasPage, hasLimit bool) (uint64, int, error) {
	page := uint64(1)
	if hasPage {
		parsed, err := parsePositiveQueryInt(pageValue)
		if err != nil {
			return 0, 0, InvalidPaginationError{Message: "page must be an integer greater than or equal to 1"}
		}
		page = parsed
	}

	limit := uint64(10)
	if hasLimit {
		parsed, err := parsePositiveQueryInt(limitValue)
		if err != nil {
			return 0, 0, InvalidPaginationError{Message: "limit must be an integer greater than or equal to 1"}
		}
		limit = parsed
	}
	if limit > 100 {
		limit = 100
	}
	return page, int(limit), nil
}

func parsePositiveQueryInt(value string) (uint64, error) {
	if value == "" {
		return 0, strconv.ErrSyntax
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, strconv.ErrSyntax
		}
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		if numberError, ok := err.(*strconv.NumError); ok && numberError.Err == strconv.ErrRange {
			// Treat arbitrarily large positive decimal input as a value beyond uint64.
			return ^uint64(0), nil
		}
		return 0, err
	}
	if parsed == 0 {
		return 0, strconv.ErrRange
	}
	return parsed, nil
}

func paginationOffset(page uint64, limit, total int) (int, bool) {
	if total <= 0 || page == 0 || limit <= 0 {
		return 0, true
	}
	pageIndex := page - 1
	// Compare before multiplying so a huge page cannot overflow the offset.
	if pageIndex > uint64(total)/uint64(limit) {
		return total, true
	}
	offset := int(pageIndex * uint64(limit))
	return offset, offset >= total
}
