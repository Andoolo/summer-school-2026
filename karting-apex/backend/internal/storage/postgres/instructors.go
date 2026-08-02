package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Instructor struct {
	ID   string
	Name string
}

type InstructorList struct {
	Items []Instructor
	Total int
}

type InstructorRepository struct {
	db *pgxpool.Pool
}

func NewInstructorRepository(db *pgxpool.Pool) *InstructorRepository {
	return &InstructorRepository{db: db}
}

func (r *InstructorRepository) List(ctx context.Context, limit, offset int) (InstructorList, error) {
	if limit == 0 {
		limit = 20
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM instructors`).Scan(&total); err != nil {
		return InstructorList{}, fmt.Errorf("count instructors: %w", err)
	}

	rows, err := r.db.Query(ctx, `
SELECT id::text, name
FROM instructors
ORDER BY name ASC
LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return InstructorList{}, fmt.Errorf("query instructors: %w", err)
	}
	defer rows.Close()

	items := make([]Instructor, 0)
	for rows.Next() {
		var instructor Instructor
		if err := rows.Scan(&instructor.ID, &instructor.Name); err != nil {
			return InstructorList{}, fmt.Errorf("scan instructor: %w", err)
		}
		items = append(items, instructor)
	}
	if err := rows.Err(); err != nil {
		return InstructorList{}, fmt.Errorf("iterate instructors: %w", err)
	}

	return InstructorList{Items: items, Total: total}, nil
}
