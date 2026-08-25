package repository

import (
	"context"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/domain"
)

type Page struct {
	Items                []domain.Equipment
	Total, Limit, Offset int
}

func (d *DB) EquipmentPage(ctx context.Context, status string, limit, offset int) (Page, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{}
	where := ""
	if status != "" {
		where = " WHERE status=?"
		args = append(args, status)
	}
	var total int
	if err := d.SQL.QueryRowContext(ctx, "SELECT count(*) FROM equipment"+where, args...).Scan(&total); err != nil {
		return Page{}, err
	}
	rows, err := d.SQL.QueryContext(ctx, "SELECT id,name,kind,capacity,status,version FROM equipment"+where+" ORDER BY name LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	items := []domain.Equipment{}
	for rows.Next() {
		var e domain.Equipment
		if err := rows.Scan(&e.ID, &e.Name, &e.Kind, &e.Capacity, &e.Status, &e.Version); err != nil {
			return Page{}, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("page equipment: %w", err)
	}
	return Page{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}
