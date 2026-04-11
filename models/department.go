package models

import (
	"database/sql"
)

type Department struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func ListDepartments(db *sql.DB) ([]Department, error) {
	rows, err := db.Query("SELECT id, name FROM departments ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []Department
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

func GetDepartment(db *sql.DB, id int) (Department, error) {
	var d Department
	err := db.QueryRow("SELECT id, name FROM departments WHERE id = $1", id).Scan(&d.ID, &d.Name)
	return d, err
}

func CreateDepartment(db *sql.DB, name string) (Department, error) {
	var d Department
	err := db.QueryRow("INSERT INTO departments (name) VALUES ($1) RETURNING id, name", name).Scan(&d.ID, &d.Name)
	return d, err
}
