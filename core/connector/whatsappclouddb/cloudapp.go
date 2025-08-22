package whatsappclouddb

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
)

type CloudRequestQuery struct {
	*dbutil.QueryHelper[*CloudRequest]
}

type CloudRequest struct {
	BusinessID      string `db:"business_phone_id"`
	WabaPhoneID     string `db:"waba_id"`
	Name            string `db:"name"`
	AdminUser       string `db:"admin_user"`
	PageAccessToken string `db:"page_access_token"`
}

const getAppByBusinessIDQuery = `
	SELECT *
	FROM wb_application
`
const insertAppQuery = `
	INSERT INTO wb_application (name, admin_user, business_phone_id, waba_id, page_access_token)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING *
`

func (cloud *CloudRequest) Scan(row dbutil.Scannable) (*CloudRequest, error) {
	err := row.Scan(
		&cloud.BusinessID,
		&cloud.WabaPhoneID,
		&cloud.Name,
		&cloud.AdminUser,
		&cloud.PageAccessToken,
	)
	if err != nil {
		return nil, err
	}
	return cloud, nil
}

func (cloud *CloudRequestQuery) SearchApp(
	ctx context.Context, waba_id string, phoneID string, name string,
) ([]*CloudRequest, error) {
	whereClauses := ""
	var args = []any{}
	argNum := 1

	if waba_id == "" && phoneID == "" {
		zerolog.Ctx(ctx).Error().Msgf("The business_id and phoneID can not be empty")
		return nil, fmt.Errorf("the business_id and phoneID can not be empty")
	}

	if waba_id != "" {
		args = append(args, waba_id)
		whereClauses += fmt.Sprintf(" WHERE waba_id = $%d", argNum)
		argNum++
	}

	if phoneID != "" && waba_id == "" {
		args = append(args, phoneID)
		whereClauses += fmt.Sprintf(" WHERE business_phone_id = $%d", argNum)
		argNum++
	}

	if phoneID != "" && waba_id != "" {
		whereClauses += fmt.Sprintf(" AND business_phone_id = $%d", argNum)
		args = append(args, phoneID)
		argNum++
	}

	if name != "" {
		whereClauses += fmt.Sprintf(" AND name = $%d", argNum)
		args = append(args, name)
		argNum++
	}

	query := getAppByBusinessIDQuery + whereClauses

	apps, err := cloud.QueryMany(ctx, query, args...)
	return apps, err
}

func (cloud *CloudRequestQuery) CreateApp(
	ctx context.Context,
	name string,
	admin_user string,
	waba_id string,
	wb_phone_id string,
	page_access_token string,
) (*CloudRequest, error) {
	cloud_insert, err := cloud.QueryOne(ctx, insertAppQuery,
		name, admin_user, wb_phone_id, waba_id, page_access_token,
	)

	return cloud_insert, err
}
