package store

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func numericPtrToFloat(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	return &f.Float64
}

func numericToFloat(n pgtype.Numeric) float64 {
	if p := numericPtrToFloat(n); p != nil {
		return *p
	}
	return 0
}

func dateOf(t time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

func datePtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return dateOf(*t)
}

func int32Ptr(p *int) *int32 {
	if p == nil {
		return nil
	}
	v := int32(*p) //nolint:gosec // bounded by the API schema
	return &v
}

func int16Ptr(p *int) *int16 {
	if p == nil {
		return nil
	}
	v := int16(*p) //nolint:gosec // bounded by the API schema
	return &v
}

func intPtr32(p *int32) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

func intPtr16(p *int16) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

type pgDate = pgtype.Date
