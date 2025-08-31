package utils

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToNumeric(val float64) pgtype.Numeric {
	var num pgtype.Numeric
	_ = num.Scan(strconv.FormatFloat(val, 'f', -1, 64)) // convert float64 -> pgtype.Numeric
	return num
}

func NumericToFloat64(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	if !f.Valid {
		return 0
	}
	return f.Float64
}

func Int32(val int32) pgtype.Int4 {
	return pgtype.Int4{
		Int32: val,
		Valid: true,
	}
}

func ToPgTimestamp(ts *timestamppb.Timestamp) pgtype.Timestamp {
	if ts == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: ts.AsTime(), Valid: true}
}

func PgToPbTimestamp(pgTs pgtype.Timestamp) *timestamppb.Timestamp {
	if !pgTs.Valid {
		return nil
	}
	return timestamppb.New(pgTs.Time)
}

func PgTextToString(pgT pgtype.Text) string {
	if pgT.Valid {
		return pgT.String
	}
	return ""
}

func StringPointerToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func PgTextToStringPointer(pgT pgtype.Text) *string {
	if pgT.Valid {
		return &pgT.String
	}
	return nil
}

func PgDateToString(pgD pgtype.Date) string {
	if pgD.Valid {
		return pgD.Time.Format("2006-01-02")
	}
	return ""
}

func PgDateToStringPointer(pgD pgtype.Date) *string {
	if pgD.Valid {
		res := pgD.Time.Format("2006-01-02")
		return &res
	}
	return nil
}
